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
	"strconv"
	"strings"
	"time"

	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/base"
	"github.com/rulego/rulego/utils/el"
	"github.com/rulego/rulego/utils/maps"
)

func init() {
	_ = rulego.Registry.Register(&ApiRouteTracerSourcegraphComponent{})
}

// ApiRouteTracerSourcegraphComponent 根据搜索路径组装 Sourcegraph 查询串并执行搜索，返回结果 JSON
type ApiRouteTracerSourcegraphComponent struct {
	Config ApiRouteTracerSourcegraphConfiguration

	endpointTpl           el.Template
	accessTokenTpl        el.Template
	repoScopeTpl          el.Template
	repoFrontendTpl       el.Template
	repoBackendTpl        el.Template
	contextGlobalTpl      el.Template
	typeFilterTpl         el.Template
	displayLimitTpl       el.Template
	defaultPatternTypeTpl el.Template
	defaultPatternsTpl    el.Template
}

type ApiRouteTracerSourcegraphConfiguration struct {
	// Sourcegraph 服务地址，例如 "https://sourcegraph.example.com"
	Endpoint string `json:"endpoint"`

	// Sourcegraph API 访问令牌，用于请求头 Authorization: token xxx
	AccessToken string `json:"accessToken"`

	// HTTP 请求超时秒数，默认 30
	TimeoutSec int `json:"timeoutSec"`

	// 仓库范围过滤，可选值:
	//   - "frontend": 仅搜索前端仓库，使用 repoFrontend 配置的正则
	//   - "backend":  仅搜索后端仓库，使用 repoBackend 配置的正则
	//   - 其他:       不限制仓库范围
	RepoScope string `json:"repoScope"`

	// 前端仓库正则，当 repoScope=frontend 时生效，默认 "teacher/fe/.*|frontend/.*"
	RepoFrontend string `json:"repoFrontend"`

	// 后端仓库正则，当 repoScope=backend 时生效，默认 "teacher/backend/.*|backend/.*"
	RepoBackend string `json:"repoBackend"`

	// 是否添加 "context:global" 查询条件，搜索所有仓库而非仅当前实例，"true"/"1"/"yes"/"on" 为真，默认 "true"
	ContextGlobal string `json:"contextGlobal"`

	// 文件类型过滤条件，直接拼接到查询串中，例如 "lang:Go" 或 "file:\.go$"
	TypeFilter string `json:"typeFilter"`

	// 搜索结果返回数量上限，对应查询串 "count:N"，默认 "1500"
	DisplayLimit string `json:"displayLimit"`

	// 默认匹配模式，可选值:
	//   - "literal": 精确匹配（默认）
	//   - "regexp":  正则匹配，添加 "patternType:regexp"
	DefaultPatternType string `json:"defaultPatternType"`

	// 默认搜索路径，换行分隔，当消息 data 未提供 patterns 时使用，例如 "/api/user/list\n/api/order/detail"
	DefaultPatterns string `json:"defaultPatterns"`
}

func (c *ApiRouteTracerSourcegraphComponent) New() types.Node {
	return &ApiRouteTracerSourcegraphComponent{Config: ApiRouteTracerSourcegraphConfiguration{
		TimeoutSec:         30,
		ContextGlobal:      "true",
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
		Desc:  "根据搜索路径组装 Sourcegraph 查询并执行搜索，返回结果 JSON",
	}
}

func (c *ApiRouteTracerSourcegraphComponent) Init(_ types.Config, configuration types.Configuration) error {
	if err := maps.Map2Struct(configuration, &c.Config); err != nil {
		return err
	}

	if c.Config.TimeoutSec <= 0 {
		c.Config.TimeoutSec = 30
	}
	if strings.TrimSpace(c.Config.ContextGlobal) == "" {
		c.Config.ContextGlobal = "true"
	}
	if strings.TrimSpace(c.Config.DisplayLimit) == "" {
		c.Config.DisplayLimit = "1500"
	}
	if strings.TrimSpace(c.Config.DefaultPatternType) == "" {
		c.Config.DefaultPatternType = "literal"
	}

	tpls := map[*el.Template]string{
		&c.endpointTpl:           c.Config.Endpoint,
		&c.accessTokenTpl:        c.Config.AccessToken,
		&c.repoScopeTpl:          c.Config.RepoScope,
		&c.repoFrontendTpl:       c.Config.RepoFrontend,
		&c.repoBackendTpl:        c.Config.RepoBackend,
		&c.contextGlobalTpl:      c.Config.ContextGlobal,
		&c.typeFilterTpl:         c.Config.TypeFilter,
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

	endpoint := strings.TrimRight(strings.TrimSpace(c.endpointTpl.ExecuteAsString(env)), "/")
	if endpoint == "" {
		ctx.TellFailure(msg, errors.New("apiRouteTracerSourcegraph: 渲染后 endpoint 为空"))
		return
	}
	accessToken := strings.TrimSpace(c.accessTokenTpl.ExecuteAsString(env))

	// --- 构建 ---
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
		ctx.TellFailure(msg, errors.New("apiRouteTracerSourcegraph: 无搜索路径，请在消息 data 传入 JSON/文本，或配置 defaultPatterns"))
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

	displayLimit := parseDisplayLimitTemplate(c.displayLimitTpl.ExecuteAsString(env), 1500)
	parts := tracerSourcegraphQueryParts{
		ContextGlobal: contextToken,
		TypeFilter:    typeFilter,
		RepoFilter:    repoFilter,
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
		ctx.TellFailure(msg, errors.New("apiRouteTracerSourcegraph: 未能生成查询串"))
		return
	}

	// --- 搜索 ---
	gqlURL, err := url.JoinPath(endpoint, ".api", "graphql")
	if err != nil {
		ctx.TellFailure(msg, fmt.Errorf("apiRouteTracerSourcegraph: 拼接 GraphQL URL 失败: %w", err))
		return
	}

	results := make([]json.RawMessage, 0, len(queries))
	for _, q := range queries {
		data, err := executeSourcegraphSearchQuery(gqlURL, accessToken, q, c.Config.TimeoutSec)
		if err != nil {
			ctx.TellFailure(msg, fmt.Errorf("apiRouteTracerSourcegraph: query %q 执行失败: %w", q, err))
			return
		}
		results = append(results, data)
	}

	// --- 输出 ---
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
	out.Metadata.PutValue("sourcegraph_search_query", first)
	if len(queries) > 1 {
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
	c.endpointTpl = nil
	c.accessTokenTpl = nil
	c.repoScopeTpl = nil
	c.repoFrontendTpl = nil
	c.repoBackendTpl = nil
	c.contextGlobalTpl = nil
	c.typeFilterTpl = nil
	c.displayLimitTpl = nil
	c.defaultPatternTypeTpl = nil
	c.defaultPatternsTpl = nil
}

func (c *ApiRouteTracerSourcegraphComponent) Close() error { return nil }

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
		return nil, errors.New("批量结果聚合失败，queries 与 results 数量不匹配")
	}
	perQuery := make([]map[string]interface{}, 0, len(queries))
	mergedResults := make([]json.RawMessage, 0)
	matchCount := 0
	limitHit := false

	for i, raw := range results {
		var data sourcegraphSearchData
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, fmt.Errorf("解析第 %d 条结果失败: %w", i+1, err)
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
