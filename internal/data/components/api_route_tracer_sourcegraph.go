package data

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/base"
	"github.com/rulego/rulego/utils/el"
	"github.com/rulego/rulego/utils/maps"
)

func init() {
	_ = rulego.Registry.Register(&ApiRouteTracerSourcegraphComponent{})
}

// ApiRouteTracerSourcegraphComponent 将 api-route-tracer 相关能力整合为一个组件：
// - gitPrepare: 克隆或拉取仓库
// - queryBuild: 组装 Sourcegraph 查询串
// - search: 调用 Sourcegraph GraphQL 搜索
type ApiRouteTracerSourcegraphComponent struct {
	Config ApiRouteTracerSourcegraphConfiguration

	actionTpl             el.Template
	gitlabURLTpl          el.Template
	workDirTpl            el.Template
	endpointTpl           el.Template
	accessTokenTpl        el.Template
	defaultSearchQueryTpl el.Template
	repoScopeTpl          el.Template
	repoFrontendTpl       el.Template
	repoBackendTpl        el.Template
	contextGlobalTpl      el.Template
	typeFilterTpl         el.Template
	includeForkedTpl      el.Template
	displayLimitTpl       el.Template
	defaultPatternTypeTpl el.Template
	defaultPatternsTpl    el.Template
}

type ApiRouteTracerSourcegraphConfiguration struct {
	// 执行动作: gitPrepare | queryBuild | search
	Action string `json:"action"`

	// gitPrepare 参数
	GitlabURL string `json:"gitlabUrl"`
	WorkDir   string `json:"workDir"`

	// search 参数
	Endpoint           string `json:"endpoint"`
	AccessToken        string `json:"accessToken"`
	TimeoutSec         int    `json:"timeoutSec"`
	DefaultSearchQuery string `json:"defaultSearchQuery"`

	// queryBuild 参数
	RepoScope          string `json:"repoScope"`
	RepoFrontend       string `json:"repoFrontend"`
	RepoBackend        string `json:"repoBackend"`
	ContextGlobal      string `json:"contextGlobal"`
	TypeFilter         string `json:"typeFilter"`
	IncludeForked      string `json:"includeForked"`
	DisplayLimit       string `json:"displayLimit"`
	DefaultPatternType string `json:"defaultPatternType"`
	DefaultPatterns    string `json:"defaultPatterns"`
}

func (c *ApiRouteTracerSourcegraphComponent) New() types.Node {
	return &ApiRouteTracerSourcegraphComponent{Config: ApiRouteTracerSourcegraphConfiguration{
		Action:             "queryBuild",
		TimeoutSec:         30,
		ContextGlobal:      "true",
		IncludeForked:      "true",
		DisplayLimit:       "1500",
		DefaultPatternType: "literal",
	}}
}

func (c *ApiRouteTracerSourcegraphComponent) Type() string {
	return "x/apiRouteTracerSourcegraph"
}

func (c *ApiRouteTracerSourcegraphComponent) Def() types.ComponentForm {
	return types.ComponentForm{
		Label: "apiRouteTracerSourcegraph",
		Desc:  "整合 gitPrepare/queryBuild/search 的 API 路由追踪组件",
	}
}

func (c *ApiRouteTracerSourcegraphComponent) Init(_ types.Config, configuration types.Configuration) error {
	if err := maps.Map2Struct(configuration, &c.Config); err != nil {
		return err
	}

	if strings.TrimSpace(c.Config.Action) == "" {
		c.Config.Action = "queryBuild"
	}
	if c.Config.TimeoutSec <= 0 {
		c.Config.TimeoutSec = 30
	}
	if strings.TrimSpace(c.Config.ContextGlobal) == "" {
		c.Config.ContextGlobal = "true"
	}
	if strings.TrimSpace(c.Config.IncludeForked) == "" {
		c.Config.IncludeForked = "true"
	}
	if strings.TrimSpace(c.Config.DisplayLimit) == "" {
		c.Config.DisplayLimit = "1500"
	}
	if strings.TrimSpace(c.Config.DefaultPatternType) == "" {
		c.Config.DefaultPatternType = "literal"
	}

	tpls := map[*el.Template]string{
		&c.actionTpl:             c.Config.Action,
		&c.gitlabURLTpl:          c.Config.GitlabURL,
		&c.workDirTpl:            c.Config.WorkDir,
		&c.endpointTpl:           c.Config.Endpoint,
		&c.accessTokenTpl:        c.Config.AccessToken,
		&c.defaultSearchQueryTpl: c.Config.DefaultSearchQuery,
		&c.repoScopeTpl:          c.Config.RepoScope,
		&c.repoFrontendTpl:       c.Config.RepoFrontend,
		&c.repoBackendTpl:        c.Config.RepoBackend,
		&c.contextGlobalTpl:      c.Config.ContextGlobal,
		&c.typeFilterTpl:         c.Config.TypeFilter,
		&c.includeForkedTpl:      c.Config.IncludeForked,
		&c.displayLimitTpl:       c.Config.DisplayLimit,
		&c.defaultPatternTypeTpl: c.Config.DefaultPatternType,
		&c.defaultPatternsTpl:    c.Config.DefaultPatterns,
	}

	for tpl, raw := range tpls {
		t, err := el.NewTemplate(strings.TrimSpace(raw))
		if err != nil {
			return err
		}
		*tpl = t
	}
	return nil
}

func (c *ApiRouteTracerSourcegraphComponent) OnMsg(ctx types.RuleContext, msg types.RuleMsg) {
	env := base.NodeUtils.GetEvnAndMetadata(ctx, msg)
	action := strings.ToLower(strings.TrimSpace(c.actionTpl.ExecuteAsString(env)))
	if action == "" {
		action = "querybuild"
	}

	switch action {
	case "gitprepare":
		c.handleGitPrepare(ctx, msg, env)
	case "querybuild":
		c.handleQueryBuild(ctx, msg, env)
	case "search":
		c.handleSearch(ctx, msg, env)
	default:
		ctx.TellFailure(msg, fmt.Errorf("apiRouteTracerSourcegraph: 不支持的 action: %s", action))
	}
}

func (c *ApiRouteTracerSourcegraphComponent) handleGitPrepare(ctx types.RuleContext, msg types.RuleMsg, env map[string]interface{}) {
	gitURL := strings.TrimSpace(c.gitlabURLTpl.ExecuteAsString(env))
	if gitURL == "" {
		ctx.TellFailure(msg, errors.New("gitPrepare: 渲染后 gitlabUrl 为空"))
		return
	}
	workDir := strings.TrimSpace(c.workDirTpl.ExecuteAsString(env))
	workDir = expandUserPath(workDir)
	workDir = filepath.Clean(workDir)
	if workDir == "" || workDir == "." {
		ctx.TellFailure(msg, errors.New("gitPrepare: 渲染后 workDir 为空"))
		return
	}

	repoBase, err := gitRepoDirNameFromURL(gitURL)
	if err != nil {
		ctx.TellFailure(msg, fmt.Errorf("gitPrepare: 从 URL 解析仓库目录名失败: %w", err))
		return
	}
	name, ok := sanitizeServiceDirName(repoBase)
	if !ok {
		ctx.TellFailure(msg, fmt.Errorf("gitPrepare: 仓库目录名非法 %q（仅允许字母、数字、.-_）", repoBase))
		return
	}
	servicePath := filepath.Join(workDir, name)

	if err := os.MkdirAll(workDir, 0o755); err != nil {
		ctx.TellFailure(msg, fmt.Errorf("gitPrepare: 创建工作目录失败: %w", err))
		return
	}

	gitEnv := append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	cloneURL := fmt.Sprintf("https://%s.git", gitURL)

	st, err := os.Stat(servicePath)
	if err == nil {
		if !st.IsDir() {
			ctx.TellFailure(msg, fmt.Errorf("gitPrepare: %q 已存在且不是目录", servicePath))
			return
		}
		if _, err := os.Stat(filepath.Join(servicePath, ".git")); err != nil {
			ctx.TellFailure(msg, fmt.Errorf("gitPrepare: %q 已存在但不是 git 仓库（缺少 .git）", servicePath))
			return
		}
		if err := runGitCommand(gitEnv, servicePath, "pull"); err != nil {
			ctx.TellFailure(msg, fmt.Errorf("gitPrepare: git pull 失败: %w", err))
			return
		}
	} else if os.IsNotExist(err) {
		if err := runGitCommand(gitEnv, workDir, "clone", cloneURL); err != nil {
			ctx.TellFailure(msg, fmt.Errorf("gitPrepare: git clone 失败: %w", err))
			return
		}
	} else {
		ctx.TellFailure(msg, fmt.Errorf("gitPrepare: 检查目录失败: %w", err))
		return
	}

	out := msg.Copy()
	if out.Metadata == nil {
		out.Metadata = types.NewMetadata()
	}
	mergeTraceMetadata(msg, out)
	out.Metadata.PutValue("api_route_tracer_service_path", servicePath)
	out.Metadata.PutValue("api_route_tracer_project_type", "")
	out.Metadata.PutValue("api_route_tracer_service_name", name)

	summary, _ := json.Marshal(map[string]string{
		"servicePath": servicePath,
		"serviceName": name,
		"gitlabUrl":   cloneURL,
	})
	out.SetData(string(summary))
	ctx.TellSuccess(out)
}

func (c *ApiRouteTracerSourcegraphComponent) handleQueryBuild(ctx types.RuleContext, msg types.RuleMsg, env map[string]interface{}) {
	patternType, patterns, fromMsg := parseQueryBuildData(msg.GetData())
	if !fromMsg {
		pt := strings.ToLower(strings.TrimSpace(c.defaultPatternTypeTpl.ExecuteAsString(env)))
		if pt != "regexp" {
			pt = "literal"
		}
		patternType = pt
		patterns = splitDefaultPatternsLines(c.defaultPatternsTpl.ExecuteAsString(env))
	}
	if len(patterns) == 0 {
		ctx.TellFailure(msg, errors.New("queryBuild: 无搜索路径，请在消息 data 传入 JSON/文本，或配置 defaultPatterns"))
		return
	}

	scope := strings.ToLower(strings.TrimSpace(c.repoScopeTpl.ExecuteAsString(env)))
	repoFE := strings.TrimSpace(c.repoFrontendTpl.ExecuteAsString(env))
	repoBE := strings.TrimSpace(c.repoBackendTpl.ExecuteAsString(env))
	repoFilter := repoFilterForTracerScope(scope, repoFE, repoBE)

	var contextToken string
	if parseTruthyTemplate(c.contextGlobalTpl.ExecuteAsString(env)) {
		contextToken = "context:global"
	}
	typeFilter := strings.TrimSpace(c.typeFilterTpl.ExecuteAsString(env))

	var forkFilter string
	if parseTruthyTemplate(c.includeForkedTpl.ExecuteAsString(env)) {
		forkFilter = "fork:yes"
	}

	displayLimit := parseDisplayLimitTemplate(c.displayLimitTpl.ExecuteAsString(env), 1500)
	parts := tracerSourcegraphQueryParts{
		ContextGlobal: contextToken,
		TypeFilter:    typeFilter,
		RepoFilter:    repoFilter,
		ForkFilter:    forkFilter,
		DisplayLimit:  displayLimit,
	}

	queries := make([]string, 0, len(patterns))
	for _, p := range patterns {
		q := buildTracerSourcegraphQuery(patternType, p, parts)
		if q != "" {
			queries = append(queries, q)
		}
	}
	if len(queries) == 0 {
		ctx.TellFailure(msg, errors.New("queryBuild: 未能生成查询串"))
		return
	}

	out := msg.Copy()
	if out.Metadata == nil {
		out.Metadata = types.NewMetadata()
	}
	mergeTraceMetadata(msg, out)

	first := queries[0]
	out.Metadata.PutValue("sourcegraph_built_query", first)
	qb, _ := json.Marshal(queries)
	out.Metadata.PutValue("sourcegraph_built_queries", string(qb))
	out.Metadata.PutValue("sourcegraph_query_repo_scope", scope)

	payload, _ := json.Marshal(map[string]interface{}{
		"query":   first,
		"queries": queries,
	})
	out.SetData(string(payload))
	ctx.TellSuccess(out)
}

func (c *ApiRouteTracerSourcegraphComponent) handleSearch(ctx types.RuleContext, msg types.RuleMsg, env map[string]interface{}) {
	endpoint := strings.TrimRight(strings.TrimSpace(c.endpointTpl.ExecuteAsString(env)), "/")
	if endpoint == "" {
		ctx.TellFailure(msg, errors.New("search: 渲染后 endpoint 为空"))
		return
	}
	accessToken := strings.TrimSpace(c.accessTokenTpl.ExecuteAsString(env))
	defaultQuery := strings.TrimSpace(c.defaultSearchQueryTpl.ExecuteAsString(env))
	queries := resolveSourcegraphQueries(msg.GetData(), defaultQuery)
	if len(queries) == 0 {
		ctx.TellFailure(msg, errors.New("search: 搜索词为空，请传入 query/queries 或配置 defaultSearchQuery"))
		return
	}

	gqlURL, err := url.JoinPath(endpoint, ".api", "graphql")
	if err != nil {
		ctx.TellFailure(msg, fmt.Errorf("search: 拼接 GraphQL URL 失败: %w", err))
		return
	}

	results := make([]json.RawMessage, 0, len(queries))
	for _, q := range queries {
		data, err := executeSourcegraphSearchQuery(gqlURL, accessToken, q, c.Config.TimeoutSec)
		if err != nil {
			ctx.TellFailure(msg, fmt.Errorf("search: query %q 执行失败: %w", q, err))
			return
		}
		results = append(results, data)
	}

	out := msg.Copy()
	if out.Metadata == nil {
		out.Metadata = types.NewMetadata()
	}
	out.Metadata.PutValue("sourcegraph_search_query", queries[0])
	if len(queries) > 1 {
		qb, _ := json.Marshal(queries)
		out.Metadata.PutValue("sourcegraph_search_queries", string(qb))
	} else {
		out.Metadata.PutValue("sourcegraph_search_queries", "")
	}

	if len(results) == 1 {
		out.SetData(string(results[0]))
		ctx.TellSuccess(out)
		return
	}

	merged, err := mergeSourcegraphSearchResults(queries, results)
	if err != nil {
		ctx.TellFailure(msg, err)
		return
	}
	out.SetData(string(merged))
	ctx.TellSuccess(out)
}

func (c *ApiRouteTracerSourcegraphComponent) Destroy() {
	c.Config = ApiRouteTracerSourcegraphConfiguration{}
	c.actionTpl = nil
	c.gitlabURLTpl = nil
	c.workDirTpl = nil
	c.endpointTpl = nil
	c.accessTokenTpl = nil
	c.defaultSearchQueryTpl = nil
	c.repoScopeTpl = nil
	c.repoFrontendTpl = nil
	c.repoBackendTpl = nil
	c.contextGlobalTpl = nil
	c.typeFilterTpl = nil
	c.includeForkedTpl = nil
	c.displayLimitTpl = nil
	c.defaultPatternTypeTpl = nil
	c.defaultPatternsTpl = nil
}

func (c *ApiRouteTracerSourcegraphComponent) Close() error { return nil }

type sourcegraphSearchRequest struct {
	Query   string   `json:"query"`
	Queries []string `json:"queries"`
}

type sourcegraphSearchResponse struct {
	Data   json.RawMessage   `json:"data"`
	Errors []json.RawMessage `json:"errors"`
}

type sourcegraphSearchData struct {
	Search *struct {
		Results *struct {
			MatchCount int               `json:"matchCount"`
			LimitHit   bool              `json:"limitHit"`
			Results    []json.RawMessage `json:"results"`
		} `json:"results"`
	} `json:"search"`
}

type sourcegraphPreprocessPatterns struct {
	PatternType string   `json:"patternType"`
	Patterns    []string `json:"patterns"`
}

type tracerSourcegraphQueryParts struct {
	ContextGlobal string
	TypeFilter    string
	RepoFilter    string
	ForkFilter    string
	DisplayLimit  int
}

const sourcegraphSearchGQL = `query RuleGoSourcegraphSearch($query: String!) {
  search(query: $query, version: V3) {
    results {
      matchCount
      limitHit
      results {
        __typename
        ... on FileMatch {
          file {
            path
            url
          }
          repository {
            name
          }
          lineMatches {
            lineNumber
            preview
          }
        }
        ... on CommitSearchResult {
          url
        }
      }
    }
  }
}`

const (
	defaultSourcegraphRepoFrontend = `teacher/fe/.*|frontend/.*`
	defaultSourcegraphRepoBackend  = `teacher/backend/.*|backend/.*`
)

func parseQueryBuildData(data string) (patternType string, patterns []string, ok bool) {
	data = strings.TrimSpace(data)
	if data == "" {
		return "", nil, false
	}
	var wrap sourcegraphPreprocessPatterns
	if err := json.Unmarshal([]byte(data), &wrap); err == nil {
		if len(wrap.Patterns) == 0 {
			return "", nil, false
		}
		pt := strings.TrimSpace(wrap.PatternType)
		if pt == "" {
			pt = "literal"
		}
		out := make([]string, 0, len(wrap.Patterns))
		for _, p := range wrap.Patterns {
			if s := strings.TrimSpace(p); s != "" {
				out = append(out, s)
			}
		}
		if len(out) == 0 {
			return "", nil, false
		}
		return pt, out, true
	}
	return "literal", []string{data}, true
}

func splitDefaultPatternsLines(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if t := strings.TrimSpace(line); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func buildTracerSourcegraphQuery(patternType, pattern string, p tracerSourcegraphQueryParts) string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return ""
	}
	pt := strings.ToLower(strings.TrimSpace(patternType))
	if pt == "" {
		pt = "literal"
	}
	var segs []string
	if strings.TrimSpace(p.ContextGlobal) != "" {
		segs = append(segs, strings.TrimSpace(p.ContextGlobal))
	}
	segs = append(segs, pattern)
	if tf := strings.TrimSpace(p.TypeFilter); tf != "" {
		segs = append(segs, tf)
	}
	if pt == "regexp" {
		segs = append(segs, "patternType:regexp")
	}
	if rf := strings.TrimSpace(p.RepoFilter); rf != "" {
		segs = append(segs, rf)
	}
	if ff := strings.TrimSpace(p.ForkFilter); ff != "" {
		segs = append(segs, ff)
	}
	s := strings.TrimSpace(strings.Join(segs, " "))
	s = strings.Join(strings.Fields(s), " ")
	if p.DisplayLimit > 0 {
		s = strings.TrimSpace(s + " " + fmt.Sprintf("count:%d", p.DisplayLimit))
	}
	return s
}

func repoFilterForTracerScope(scope, repoFrontend, repoBackend string) string {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "frontend":
		rf := strings.TrimSpace(repoFrontend)
		if rf == "" {
			rf = defaultSourcegraphRepoFrontend
		}
		return "repo:(" + rf + ")"
	case "backend":
		rb := strings.TrimSpace(repoBackend)
		if rb == "" {
			rb = defaultSourcegraphRepoBackend
		}
		return "repo:(" + rb + ")"
	default:
		return ""
	}
}

func parseTruthyTemplate(s string) bool {
	v := strings.ToLower(strings.TrimSpace(s))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func parseDisplayLimitTemplate(s string, fallback int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func resolveSourcegraphQueries(data string, defaultQ string) []string {
	data = strings.TrimSpace(data)
	if data == "" {
		if q := strings.TrimSpace(defaultQ); q != "" {
			return []string{q}
		}
		return nil
	}
	var wrap sourcegraphSearchRequest
	if err := json.Unmarshal([]byte(data), &wrap); err == nil {
		out := make([]string, 0, len(wrap.Queries)+1)
		for _, q := range wrap.Queries {
			if s := strings.TrimSpace(q); s != "" {
				out = append(out, s)
			}
		}
		if len(out) > 0 {
			return out
		}
		if q := strings.TrimSpace(wrap.Query); q != "" {
			return []string{q}
		}
		if q := strings.TrimSpace(defaultQ); q != "" {
			return []string{q}
		}
		return nil
	}
	return []string{data}
}

func executeSourcegraphSearchQuery(gqlURL, accessToken, query string, timeoutSec int) (json.RawMessage, error) {
	payload := map[string]interface{}{
		"query": sourcegraphSearchGQL,
		"variables": map[string]string{
			"query": query,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	ctxHTTP, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctxHTTP, http.MethodPost, gqlURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if accessToken != "" {
		req.Header.Set("Authorization", "token "+accessToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncateForTracerLog(strings.TrimSpace(string(respBody)), 512))
	}

	var gqlResp sourcegraphSearchResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		preview := truncateForTracerLog(strings.TrimSpace(string(respBody)), 320)
		return nil, fmt.Errorf("响应非 JSON，正文片段: %q: %w", preview, err)
	}
	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL 错误: %s", string(gqlResp.Errors[0]))
	}
	return gqlResp.Data, nil
}

func mergeSourcegraphSearchResults(queries []string, results []json.RawMessage) ([]byte, error) {
	if len(queries) == 0 || len(results) == 0 || len(queries) != len(results) {
		return nil, errors.New("search: 批量结果聚合失败，queries 与 results 数量不匹配")
	}
	perQuery := make([]map[string]interface{}, 0, len(queries))
	mergedResults := make([]json.RawMessage, 0)
	matchCount := 0
	limitHit := false

	for i, raw := range results {
		var data sourcegraphSearchData
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, fmt.Errorf("search: 解析第 %d 条结果失败: %w", i+1, err)
		}
		perQuery = append(perQuery, map[string]interface{}{
			"query": queries[i],
			"data":  json.RawMessage(raw),
		})
		if data.Search == nil || data.Search.Results == nil {
			continue
		}
		matchCount += data.Search.Results.MatchCount
		if data.Search.Results.LimitHit {
			limitHit = true
		}
		mergedResults = append(mergedResults, data.Search.Results.Results...)
	}

	payload := map[string]interface{}{
		"query":   queries[0],
		"queries": queries,
		"search": map[string]interface{}{
			"results": map[string]interface{}{
				"matchCount": matchCount,
				"limitHit":   limitHit,
				"results":    mergedResults,
			},
		},
		"results": perQuery,
	}
	return json.Marshal(payload)
}

func truncateForTracerLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func expandUserPath(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	if s == "~" {
		h, err := os.UserHomeDir()
		if err == nil && h != "" {
			return h
		}
		return s
	}
	if strings.HasPrefix(s, "~/") {
		h, err := os.UserHomeDir()
		if err == nil && h != "" {
			return filepath.Join(h, strings.TrimPrefix(s, "~/"))
		}
	}
	return s
}

func gitRepoDirNameFromURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("url 为空")
	}
	switch {
	case strings.HasPrefix(raw, "ssh://"):
		u, err := url.Parse(raw)
		if err != nil {
			return "", err
		}
		path := strings.TrimSuffix(strings.Trim(u.Path, "/"), ".git")
		if path == "" {
			return "", errors.New("ssh:// URL 路径为空")
		}
		parts := strings.Split(path, "/")
		return parts[len(parts)-1], nil
	case strings.Contains(raw, "://"):
		u, err := url.Parse(raw)
		if err != nil {
			return "", err
		}
		path := strings.TrimSuffix(strings.Trim(u.Path, "/"), ".git")
		parts := strings.Split(path, "/")
		if len(parts) == 0 || parts[len(parts)-1] == "" {
			return "", errors.New("无法从 URL 路径得到仓库名")
		}
		return parts[len(parts)-1], nil
	case strings.HasPrefix(raw, "git@"):
		colon := strings.Index(raw, ":")
		if colon < 0 {
			return "", errors.New("SCP 形式 git URL 中缺少 ':'")
		}
		path := strings.TrimSuffix(strings.Trim(raw[colon+1:], "/"), ".git")
		parts := strings.Split(path, "/")
		if len(parts) == 0 || parts[len(parts)-1] == "" {
			return "", errors.New("无法从 SCP 形式 URL 得到仓库名")
		}
		return parts[len(parts)-1], nil
	default:
		path := strings.TrimSuffix(strings.Trim(raw, "/"), ".git")
		parts := strings.Split(path, "/")
		if len(parts) == 0 || parts[len(parts)-1] == "" {
			return "", errors.New("无法解析为非空仓库名")
		}
		return parts[len(parts)-1], nil
	}
}

func sanitizeServiceDirName(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" || len(s) > 128 {
		return "", false
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			continue
		}
		return "", false
	}
	return s, true
}

func runGitCommand(env []string, dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = env
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %v: %w: %s", args, err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func mergeTraceMetadata(from, to types.RuleMsg) {
	if from.Metadata == nil || to.Metadata == nil {
		return
	}
	for _, key := range []string{
		"trace_url", "trace_method", "trace_router_http_status",
		"api_route_tracer_service_index",
	} {
		if v := from.Metadata.GetValue(key); v != "" {
			to.Metadata.PutValue(key, v)
		}
	}
}
