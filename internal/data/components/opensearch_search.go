package data

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/base"
	"github.com/rulego/rulego/utils/el"
	"github.com/rulego/rulego/utils/maps"
)

// OpenSearchSearchNode 调用 OpenSearch / Elasticsearch `_search` 检索日志（POST JSON）。
type OpenSearchSearchNode struct {
	cfg                 openSearchSearchConfig
	endpointTemplate    el.Template
	indexTemplate       el.Template
	defaultBodyTemplate el.Template
}

type openSearchSearchConfig struct {
	Endpoint           string `json:"endpoint"`
	Index              string `json:"index"`
	Username           string `json:"username"`
	Password           string `json:"password"`
	InsecureSkipVerify bool   `json:"insecureSkipVerify"`
	TimeoutSec         int    `json:"timeoutSec"`
	SearchType         string `json:"searchType"`
	IgnoreUnavailable  bool   `json:"ignoreUnavailable"`
	DefaultSearchBody  string `json:"defaultSearchBody"`
}

func (n *OpenSearchSearchNode) Type() string { return "opensearch/search" }

func (n *OpenSearchSearchNode) New() types.Node { return &OpenSearchSearchNode{} }

func (n *OpenSearchSearchNode) Def() types.ComponentForm {
	return types.ComponentForm{
		Label: "opensearchSearch",
		Desc:  "OpenSearch/ES _search 日志检索",
	}
}

var openSearchIndexPattern = regexp.MustCompile(`^[a-zA-Z0-9_*,.-]+$`)

func (n *OpenSearchSearchNode) Init(_ types.Config, configuration types.Configuration) error {
	if err := maps.Map2Struct(configuration, &n.cfg); err != nil {
		return err
	}
	n.cfg.Endpoint = strings.TrimRight(strings.TrimSpace(n.cfg.Endpoint), "/")
	n.cfg.Index = strings.TrimSpace(n.cfg.Index)
	n.cfg.Username = strings.TrimSpace(n.cfg.Username)
	n.cfg.Password = strings.TrimSpace(n.cfg.Password)
	n.cfg.DefaultSearchBody = strings.TrimSpace(n.cfg.DefaultSearchBody)
	n.cfg.SearchType = strings.TrimSpace(n.cfg.SearchType)
	if n.cfg.Endpoint == "" {
		return errors.New("opensearch/search: endpoint 不能为空（如 https://opensearch:9200）")
	}
	if n.cfg.TimeoutSec <= 0 {
		n.cfg.TimeoutSec = 60
	}
	if n.cfg.SearchType == "" {
		n.cfg.SearchType = "query_then_fetch"
	}
	if n.cfg.SearchType != "query_then_fetch" && n.cfg.SearchType != "dfs_query_then_fetch" {
		return fmt.Errorf("opensearch/search: searchType 非法: %q（仅支持 query_then_fetch / dfs_query_then_fetch）", n.cfg.SearchType)
	}
	if n.cfg.DefaultSearchBody == "" {
		n.cfg.DefaultSearchBody = `{"size":100,"sort":[{"@timestamp":{"order":"desc"}}],"query":{"match_all":{}}}`
	}
	if !json.Valid([]byte(n.cfg.DefaultSearchBody)) {
		return errors.New("opensearch/search: defaultSearchBody 不是合法 JSON")
	}
	var err error
	n.endpointTemplate, err = el.NewTemplate(n.cfg.Endpoint)
	if err != nil {
		return fmt.Errorf("opensearch/search: endpoint 模板非法: %w", err)
	}
	n.indexTemplate, err = el.NewTemplate(n.cfg.Index)
	if err != nil {
		return fmt.Errorf("opensearch/search: index 模板非法: %w", err)
	}
	n.defaultBodyTemplate, err = el.NewTemplate(n.cfg.DefaultSearchBody)
	if err != nil {
		return fmt.Errorf("opensearch/search: defaultSearchBody 模板非法: %w", err)
	}
	return nil
}

var openSearchSearchBodyTopKeys = map[string]struct{}{
	"query": {}, "post_filter": {}, "size": {}, "from": {}, "sort": {},
	"track_total_hits": {}, "_source": {}, "stored_fields": {}, "script_fields": {},
	"docvalue_fields": {}, "fields": {}, "highlight": {}, "rescore": {}, "suggest": {},
	"aggregations": {}, "aggs": {}, "collapse": {}, "version": {}, "explain": {},
	"profile": {}, "timeout": {}, "min_score": {}, "search_after": {}, "pit": {},
	"runtime_mappings": {}, "indices_boost": {}, "seq_no_primary_term": {},
	"track_scores": {}, "terminate_after": {}, "stats": {}, "batched_reduce_size": {},
}

func isPlausibleOpenSearchSearchBody(m map[string]interface{}) bool {
	for k := range m {
		if _, ok := openSearchSearchBodyTopKeys[k]; !ok {
			return false
		}
	}
	if q, ok := m["query"]; ok {
		_, isObj := q.(map[string]interface{})
		if !isObj {
			return false
		}
	}
	return true
}

func mergeMsgPayloadIntoDefaultSearch(defaultJSON string, msgObj map[string]interface{}) ([]byte, error) {
	var def map[string]interface{}
	if err := json.Unmarshal([]byte(defaultJSON), &def); err != nil {
		def = map[string]interface{}{
			"size": 100,
			"sort": []interface{}{
				map[string]interface{}{"@timestamp": map[string]interface{}{"order": "desc"}},
			},
			"query": map[string]interface{}{"match_all": map[string]interface{}{}},
		}
	}
	baseQ, ok := def["query"]
	if !ok {
		baseQ = map[string]interface{}{"match_all": map[string]interface{}{}}
	}
	var extraMust []interface{}
	for k, v := range msgObj {
		switch val := v.(type) {
		case string:
			s := strings.TrimSpace(val)
			if s != "" {
				extraMust = append(extraMust, map[string]interface{}{
					"match": map[string]interface{}{k: s},
				})
			}
		case float64:
			extraMust = append(extraMust, map[string]interface{}{
				"term": map[string]interface{}{k: val},
			})
		case bool:
			extraMust = append(extraMust, map[string]interface{}{
				"term": map[string]interface{}{k: val},
			})
		case nil:
			continue
		default:
			extraMust = append(extraMust, map[string]interface{}{
				"match": map[string]interface{}{k: fmt.Sprint(val)},
			})
		}
	}
	if len(extraMust) == 0 {
		return json.Marshal(def)
	}
	def["query"] = map[string]interface{}{
		"bool": map[string]interface{}{
			"must": append([]interface{}{baseQ}, extraMust...),
		},
	}
	return json.Marshal(def)
}

func validateOpenSearchIndex(index string) error {
	idx := strings.TrimSpace(index)
	if idx == "" {
		return errors.New("opensearch/search: index 不能为空（支持单索引、逗号分隔多索引或通配）")
	}
	for _, part := range strings.Split(idx, ",") {
		p := strings.TrimSpace(part)
		if p == "" || !openSearchIndexPattern.MatchString(p) {
			return fmt.Errorf("opensearch/search: index 片段非法: %q（仅允许字母数字及 _ * , . -）", part)
		}
	}
	return nil
}

func resolveOpenSearchBody(msgData string, defaultJSON string) ([]byte, error) {
	msgData = strings.TrimSpace(msgData)
	if msgData == "" {
		return normalizeQueryStringInBody([]byte(defaultJSON)), nil
	}
	if json.Valid([]byte(msgData)) {
		var probe interface{}
		if err := json.Unmarshal([]byte(msgData), &probe); err != nil {
			return nil, err
		}
		if obj, isObj := probe.(map[string]interface{}); isObj {
			if isPlausibleOpenSearchSearchBody(obj) {
				return normalizeQueryStringInBody([]byte(msgData)), nil
			}
			merged, err := mergeMsgPayloadIntoDefaultSearch(defaultJSON, obj)
			if err != nil {
				return nil, err
			}
			return normalizeQueryStringInBody(merged), nil
		}
	}
	wrap := map[string]interface{}{
		"size": 100,
		"sort": []interface{}{
			map[string]interface{}{"@timestamp": map[string]interface{}{"order": "desc"}},
		},
		"query": map[string]interface{}{
			"query_string": map[string]interface{}{
				"query": msgData,
			},
		},
	}
	var def map[string]interface{}
	if err := json.Unmarshal([]byte(defaultJSON), &def); err == nil {
		if sz, ok := def["size"].(float64); ok && sz > 0 {
			wrap["size"] = int(sz)
		}
		if s, ok := def["sort"]; ok {
			wrap["sort"] = s
		}
	}
	out, err := json.Marshal(wrap)
	if err != nil {
		return nil, err
	}
	return normalizeQueryStringInBody(out), nil
}

func isJSONObjectString(raw string) bool {
	s := strings.TrimSpace(raw)
	if s == "" || !json.Valid([]byte(s)) {
		return false
	}
	var probe interface{}
	if err := json.Unmarshal([]byte(s), &probe); err != nil {
		return false
	}
	_, ok := probe.(map[string]interface{})
	return ok
}

func normalizeQueryStringInBody(body []byte) []byte {
	var root interface{}
	if err := json.Unmarshal(body, &root); err != nil {
		return body
	}
	changed := false
	var walk func(v interface{})
	normalizeQSMap := func(qs map[string]interface{}) {
		qRaw, ok := qs["query"]
		if !ok {
			return
		}
		qStr, ok := queryFieldToString(qRaw)
		if !ok {
			return
		}
		trimmed := strings.TrimSpace(qStr)
		if shouldQuoteAsLiteralQueryString(trimmed) {
			qs["query"] = quoteAsPhrase(trimmed)
			changed = true
		}
	}
	walk = func(v interface{}) {
		switch cur := v.(type) {
		case map[string]interface{}:
			for _, key := range []string{"query_string", "simple_query_string"} {
				if qsRaw, ok := cur[key]; ok {
					if qs, ok := qsRaw.(map[string]interface{}); ok {
						normalizeQSMap(qs)
					}
				}
			}
			for _, vv := range cur {
				walk(vv)
			}
		case []interface{}:
			for _, vv := range cur {
				walk(vv)
			}
		}
	}
	walk(root)
	if !changed {
		return body
	}
	out, err := json.Marshal(root)
	if err != nil {
		return body
	}
	return out
}

func queryFieldToString(v interface{}) (string, bool) {
	switch x := v.(type) {
	case string:
		return x, true
	case float64:
		return strings.TrimSpace(fmt.Sprint(x)), true
	case json.Number:
		return x.String(), true
	case bool:
		if x {
			return "true", true
		}
		return "false", true
	default:
		return "", false
	}
}

func shouldQuoteAsLiteralQueryString(s string) bool {
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
		return false
	}
	upper := strings.ToUpper(s)
	if strings.Contains(upper, " AND ") || strings.Contains(upper, " OR ") || strings.Contains(upper, " NOT ") {
		return false
	}
	if strings.ContainsAny(s, "()[]{}^~*?") {
		return false
	}
	if strings.HasPrefix(s, "/") {
		return true
	}
	if strings.Contains(s, "/") {
		return true
	}
	return false
}

func quoteAsPhrase(s string) string {
	escaped := strings.ReplaceAll(s, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	return `"` + escaped + `"`
}

func openSearchSearchURL(endpoint, index, searchType string, ignoreUnavailable bool) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" || u.Host == "" {
		return "", errors.New("endpoint 需包含协议与主机，如 https://host:9200")
	}
	idx := strings.TrimSpace(index)
	basePath := strings.TrimSuffix(u.Path, "/")
	var path string
	if basePath == "" {
		path = "/" + idx + "/_search"
	} else {
		path = basePath + "/" + idx + "/_search"
	}
	u.Path = path
	q := u.Query()
	if strings.TrimSpace(searchType) != "" {
		q.Set("search_type", strings.TrimSpace(searchType))
	}
	q.Set("ignore_unavailable", fmt.Sprintf("%t", ignoreUnavailable))
	u.RawQuery = q.Encode()
	u.Fragment = ""
	return u.String(), nil
}

func (n *OpenSearchSearchNode) OnMsg(ctx types.RuleContext, msg types.RuleMsg) {
	env := base.NodeUtils.GetEvnAndMetadata(ctx, msg)
	endpoint := strings.TrimSpace(n.endpointTemplate.ExecuteAsString(env))
	index := strings.TrimSpace(n.indexTemplate.ExecuteAsString(env))
	defaultSearchBody := strings.TrimSpace(n.defaultBodyTemplate.ExecuteAsString(env))

	if endpoint == "" {
		ctx.TellFailure(msg, errors.New("opensearch/search: 渲染后 endpoint 为空"))
		return
	}
	if err := validateOpenSearchIndex(index); err != nil {
		ctx.TellFailure(msg, err)
		return
	}
	if defaultSearchBody == "" {
		defaultSearchBody = `{"size":100,"sort":[{"@timestamp":{"order":"desc"}}],"query":{"match_all":{}}}`
	}
	if !json.Valid([]byte(defaultSearchBody)) {
		ctx.TellFailure(msg, errors.New("opensearch/search: 渲染后 defaultSearchBody 不是合法 JSON"))
		return
	}

	msgData := msg.GetData()
	if strings.Contains(n.cfg.DefaultSearchBody, "${msg.") && isJSONObjectString(msgData) {
		msgData = ""
	}
	body, err := resolveOpenSearchBody(msgData, defaultSearchBody)
	if err != nil {
		ctx.TellFailure(msg, fmt.Errorf("opensearch/search: 构造请求体失败: %w", err))
		return
	}
	searchURL, err := openSearchSearchURL(endpoint, index, n.cfg.SearchType, n.cfg.IgnoreUnavailable)
	if err != nil {
		ctx.TellFailure(msg, fmt.Errorf("opensearch/search: 拼接 URL 失败: %w", err))
		return
	}

	transport := http.RoundTripper(http.DefaultTransport)
	if n.cfg.InsecureSkipVerify {
		if baseTransport, ok := http.DefaultTransport.(*http.Transport); ok {
			tr := baseTransport.Clone()
			if tr.TLSClientConfig == nil {
				tr.TLSClientConfig = &tls.Config{}
			}
			tr.TLSClientConfig.InsecureSkipVerify = true
			transport = tr
		}
	}
	client := &http.Client{
		Timeout:   time.Duration(n.cfg.TimeoutSec) * time.Second,
		Transport: transport,
	}

	ctxHTTP, cancel := context.WithTimeout(context.Background(), time.Duration(n.cfg.TimeoutSec)*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctxHTTP, http.MethodPost, searchURL, bytes.NewReader(body))
	if err != nil {
		ctx.TellFailure(msg, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if n.cfg.Username != "" || n.cfg.Password != "" {
		req.SetBasicAuth(n.cfg.Username, n.cfg.Password)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[rulego] opensearch/search 请求失败: %v", err)
		ctx.TellFailure(msg, err)
		return
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		ctx.TellFailure(msg, err)
		return
	}

	out := msg.Copy()
	if out.Metadata == nil {
		out.Metadata = types.NewMetadata()
	}
	out.Metadata.PutValue("opensearch_index", index)
	out.Metadata.PutValue("opensearch_http_status", fmt.Sprintf("%d", resp.StatusCode))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		ctx.TellFailure(msg, fmt.Errorf("opensearch/search: HTTP %d: %s", resp.StatusCode, truncateForLog(string(respBody), 1024)))
		return
	}

	var summary struct {
		Hits *struct {
			Total interface{} `json:"total"`
		} `json:"hits"`
		Took int `json:"took"`
	}
	if err := json.Unmarshal(respBody, &summary); err == nil && summary.Hits != nil {
		out.Metadata.PutValue("opensearch_took_ms", fmt.Sprintf("%d", summary.Took))
		switch t := summary.Hits.Total.(type) {
		case float64:
			out.Metadata.PutValue("opensearch_hits_total", fmt.Sprintf("%.0f", t))
		case map[string]interface{}:
			if v, ok := t["value"]; ok {
				out.Metadata.PutValue("opensearch_hits_total", fmt.Sprintf("%v", v))
			}
		}
	}

	out.SetData(string(respBody))
	ctx.TellSuccess(out)
}

func (n *OpenSearchSearchNode) Destroy() {
	n.cfg = openSearchSearchConfig{}
	n.endpointTemplate = nil
	n.indexTemplate = nil
	n.defaultBodyTemplate = nil
}

func truncateForLog(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}

func init() {
	_ = rulego.Registry.Register(&OpenSearchSearchNode{})
}
