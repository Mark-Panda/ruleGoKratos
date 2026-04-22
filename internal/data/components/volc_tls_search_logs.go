package data

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/base"
	"github.com/rulego/rulego/utils/el"
	"github.com/rulego/rulego/utils/maps"
	"github.com/volcengine/volc-sdk-golang/service/tls"
)

// VolcTlsSearchLogsNode 调用火山引擎日志服务 TLS SearchLogs / SearchLogsV2 检索日志。
type VolcTlsSearchLogsNode struct {
	cfg              volcTlsSearchLogsConfig
	defaultQueryTmpl el.Template
}

type volcTlsSearchLogsConfig struct {
	Endpoint           string `json:"endpoint"`
	Region             string `json:"region"`
	AccessKeyID        string `json:"accessKeyId"`
	SecretAccessKey    string `json:"secretAccessKey"`
	SessionToken       string `json:"sessionToken"`
	TopicID            string `json:"topicId"`
	DefaultQuery       string `json:"defaultQuery"`
	Limit              int    `json:"limit"`
	UseAPIV3           bool   `json:"useApiV3"`
	TimeoutSec         int    `json:"timeoutSec"`
	TimeRangePreset    string `json:"timeRangePreset"`
	DefaultStartTimeMs int64  `json:"defaultStartTimeMs"`
	DefaultEndTimeMs   int64  `json:"defaultEndTimeMs"`
	DefaultSort        string `json:"defaultSort"`
	HighLight          bool   `json:"highLight"`
}

func (n *VolcTlsSearchLogsNode) Type() string { return "volcTls/searchLogs" }

func (n *VolcTlsSearchLogsNode) New() types.Node { return &VolcTlsSearchLogsNode{} }

func (n *VolcTlsSearchLogsNode) Def() types.ComponentForm {
	return types.ComponentForm{
		Label: "volcTlsSearchLogs",
		Desc:  "火山 TLS 检索日志（SearchLogs / SearchLogsV2）",
	}
}

func (n *VolcTlsSearchLogsNode) Init(_ types.Config, configuration types.Configuration) error {
	if err := maps.Map2Struct(configuration, &n.cfg); err != nil {
		return err
	}
	n.cfg.Endpoint = strings.TrimSpace(n.cfg.Endpoint)
	n.cfg.Region = strings.TrimSpace(n.cfg.Region)
	n.cfg.AccessKeyID = strings.TrimSpace(n.cfg.AccessKeyID)
	n.cfg.SecretAccessKey = strings.TrimSpace(n.cfg.SecretAccessKey)
	n.cfg.SessionToken = strings.TrimSpace(n.cfg.SessionToken)
	n.cfg.TopicID = strings.TrimSpace(n.cfg.TopicID)
	n.cfg.DefaultQuery = strings.TrimSpace(n.cfg.DefaultQuery)
	if n.cfg.DefaultQuery == "" {
		n.cfg.DefaultQuery = "*"
	}
	if n.cfg.Limit <= 0 {
		n.cfg.Limit = 100
	}
	if n.cfg.Limit > 500 {
		n.cfg.Limit = 500
	}
	if n.cfg.TimeoutSec <= 0 {
		n.cfg.TimeoutSec = 60
	}
	n.cfg.TimeRangePreset = strings.TrimSpace(n.cfg.TimeRangePreset)
	n.cfg.DefaultSort = strings.TrimSpace(n.cfg.DefaultSort)
	if n.cfg.DefaultSort == "" {
		n.cfg.DefaultSort = "desc"
	}
	if n.cfg.DefaultSort != "asc" && n.cfg.DefaultSort != "desc" {
		n.cfg.DefaultSort = "desc"
	}
	if n.cfg.AccessKeyID == "" || n.cfg.SecretAccessKey == "" {
		return errors.New("volcTls/searchLogs: accessKeyId 与 secretAccessKey 不能为空")
	}
	if n.cfg.Region == "" {
		return errors.New("volcTls/searchLogs: region 不能为空（如 cn-beijing）")
	}
	if n.cfg.TopicID == "" {
		return errors.New("volcTls/searchLogs: topicId 不能为空")
	}
	var err error
	n.defaultQueryTmpl, err = el.NewTemplate(n.cfg.DefaultQuery)
	if err != nil {
		return fmt.Errorf("volcTls/searchLogs: defaultQuery 模板: %w", err)
	}
	return nil
}

type volcTlsSearchMsg struct {
	Query     string `json:"query"`
	TLSQuery  string `json:"tlsQuery"`
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
	TopicID   string `json:"topicId"`
	Context   string `json:"context"`
	Sort      string `json:"sort"`
	HighLight *bool  `json:"highLight"`
}

func tlsDefaultTimeWindowMS(cfg *volcTlsSearchLogsConfig, nowMs int64) (startMs, endMs int64) {
	if cfg == nil {
		return nowMs - 15*60*1000, nowMs
	}
	preset := strings.TrimSpace(cfg.TimeRangePreset)
	if preset == "" {
		preset = "last_15m"
	}
	endMs = nowMs
	switch preset {
	case "custom":
		s, e := cfg.DefaultStartTimeMs, cfg.DefaultEndTimeMs
		if e > s && s > 0 {
			return s, e
		}
		return nowMs - 15*60*1000, nowMs
	case "last_30m":
		return nowMs - 30*60*1000, nowMs
	case "last_1h":
		return nowMs - 60*60*1000, nowMs
	case "last_6h":
		return nowMs - 6*60*60*1000, nowMs
	case "last_24h":
		return nowMs - 24*60*60*1000, nowMs
	case "last_7d":
		return nowMs - 7*24*60*60*1000, nowMs
	case "today_local":
		t := time.UnixMilli(nowMs).In(time.Local)
		startOfDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
		return startOfDay.UnixMilli(), nowMs
	default:
		return nowMs - 15*60*1000, nowMs
	}
}

func resolveVolcTlsSearch(msgData string, topicDefault, queryDefault string, cfg *volcTlsSearchLogsConfig) (volcTlsSearchMsg, error) {
	now := time.Now().UnixMilli()
	startDef, endDef := tlsDefaultTimeWindowMS(cfg, now)
	sortDef := "desc"
	if cfg != nil {
		sortDef = strings.TrimSpace(cfg.DefaultSort)
	}
	if sortDef == "" {
		sortDef = "desc"
	}
	if sortDef != "asc" && sortDef != "desc" {
		sortDef = "desc"
	}
	out := volcTlsSearchMsg{
		Query:     queryDefault,
		StartTime: startDef,
		EndTime:   endDef,
		TopicID:   topicDefault,
		Sort:      sortDef,
	}
	msgData = strings.TrimSpace(msgData)
	if msgData == "" {
		return out, nil
	}
	var raw volcTlsSearchMsg
	if err := json.Unmarshal([]byte(msgData), &raw); err != nil {
		out.Query = msgData
		return out, nil
	}
	if q := strings.TrimSpace(raw.Query); q != "" {
		out.Query = q
	} else if q := strings.TrimSpace(raw.TLSQuery); q != "" {
		out.Query = q
	}
	if raw.StartTime > 0 {
		out.StartTime = raw.StartTime
	}
	if raw.EndTime > 0 {
		out.EndTime = raw.EndTime
	}
	if strings.TrimSpace(raw.TopicID) != "" {
		out.TopicID = strings.TrimSpace(raw.TopicID)
	}
	if raw.Context != "" {
		out.Context = raw.Context
	}
	if strings.TrimSpace(raw.Sort) != "" {
		out.Sort = strings.TrimSpace(raw.Sort)
	}
	if raw.HighLight != nil {
		out.HighLight = raw.HighLight
	} else if cfg != nil && cfg.HighLight {
		v := true
		out.HighLight = &v
	}
	if out.TopicID == "" {
		return volcTlsSearchMsg{}, errors.New("volcTls/searchLogs: topicId 为空（请在节点配置或消息 JSON 中提供 topicId）")
	}
	if out.Query == "" {
		out.Query = "*"
	}
	if out.EndTime < out.StartTime {
		return volcTlsSearchMsg{}, errors.New("volcTls/searchLogs: endTime 不能小于 startTime")
	}
	return out, nil
}

func tlsEndpointFor(cfg *volcTlsSearchLogsConfig) string {
	if cfg.Endpoint != "" {
		return strings.TrimRight(cfg.Endpoint, "/")
	}
	return fmt.Sprintf("https://tls.%s.volces.com", cfg.Region)
}

func newVolcTLSHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: 50,
			IdleConnTimeout:     10 * time.Second,
			DisableCompression:  false,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 10 * time.Second,
			}).DialContext,
		},
	}
}

func decodeVolcTLSBadResponseError(err error) error {
	var bre *tls.BadResponseError
	if err == nil || !errors.As(err, &bre) {
		return err
	}
	body := []byte(bre.RespBody)
	if len(body) >= 2 && body[0] == 0x1f && body[1] == 0x8b {
		gr, e := gzip.NewReader(bytes.NewReader(body))
		if e == nil {
			defer gr.Close()
			if dec, e := io.ReadAll(gr); e == nil {
				body = dec
			}
		}
	}
	var api tls.Error
	if json.Unmarshal(body, &api) == nil && (api.Message != "" || api.Code != "") {
		httpC := api.HTTPCode
		if httpC == 0 {
			httpC = int32(bre.HTTPCode)
		}
		rid := api.RequestID
		if rid == "" && bre.RespHeader != nil {
			if vv := bre.RespHeader["X-Tls-Requestid"]; len(vv) > 0 {
				rid = vv[0]
			}
		}
		return fmt.Errorf("volcTls/searchLogs: HTTP %d [%s] %s (requestID=%s)", httpC, api.Code, api.Message, rid)
	}
	s := strings.TrimSpace(string(body))
	if s != "" && len(s) < 4096 {
		return fmt.Errorf("volcTls/searchLogs: HTTP %d %s", bre.HTTPCode, s)
	}
	return err
}

func (n *VolcTlsSearchLogsNode) OnMsg(ctx types.RuleContext, msg types.RuleMsg) {
	env := base.NodeUtils.GetEvnAndMetadata(ctx, msg)
	queryDefault := strings.TrimSpace(n.defaultQueryTmpl.ExecuteAsString(env))
	if queryDefault == "" {
		queryDefault = "*"
	}
	params, err := resolveVolcTlsSearch(msg.GetData(), n.cfg.TopicID, queryDefault, &n.cfg)
	if err != nil {
		ctx.TellFailure(msg, err)
		return
	}

	client := tls.NewClient(
		tlsEndpointFor(&n.cfg),
		n.cfg.AccessKeyID,
		n.cfg.SecretAccessKey,
		n.cfg.SessionToken,
		n.cfg.Region,
	)
	to := time.Duration(n.cfg.TimeoutSec) * time.Second
	if err := client.SetHttpClient(newVolcTLSHTTPClient(to)); err != nil {
		ctx.TellFailure(msg, err)
		return
	}
	client.SetTimeout(to)

	req := &tls.SearchLogsRequest{
		TopicID:   params.TopicID,
		Query:     params.Query,
		StartTime: params.StartTime,
		EndTime:   params.EndTime,
		Limit:     n.cfg.Limit,
		Context:   params.Context,
		Sort:      params.Sort,
	}
	if params.HighLight != nil {
		req.HighLight = *params.HighLight
	}

	ctxCall, cancel := context.WithTimeout(context.Background(), time.Duration(n.cfg.TimeoutSec)*time.Second)
	defer cancel()

	type searchResult struct {
		resp *tls.SearchLogsResponse
		err  error
	}
	ch := make(chan searchResult, 1)
	go func() {
		var resp *tls.SearchLogsResponse
		var e error
		if n.cfg.UseAPIV3 {
			resp, e = client.SearchLogsV2(req)
		} else {
			resp, e = client.SearchLogs(req)
		}
		ch <- searchResult{resp, e}
	}()

	var resp *tls.SearchLogsResponse
	select {
	case <-ctxCall.Done():
		log.Printf("[rulego] volcTls/searchLogs 超时: %v", ctxCall.Err())
		ctx.TellFailure(msg, fmt.Errorf("volcTls/searchLogs: 请求超时（%ds）", n.cfg.TimeoutSec))
		return
	case r := <-ch:
		resp, err = r.resp, r.err
	}
	if err != nil {
		err = decodeVolcTLSBadResponseError(err)
		log.Printf("[rulego] volcTls/searchLogs 失败: %v", err)
		ctx.TellFailure(msg, err)
		return
	}

	outBody, err := json.Marshal(resp)
	if err != nil {
		ctx.TellFailure(msg, err)
		return
	}
	out := msg.Copy()
	if out.Metadata == nil {
		out.Metadata = types.NewMetadata()
	}
	out.Metadata.PutValue("volc_tls_topic_id", params.TopicID)
	out.Metadata.PutValue("volc_tls_query", params.Query)
	out.Metadata.PutValue("volc_tls_hit_count", fmt.Sprintf("%d", resp.HitCount))
	out.SetData(string(outBody))
	ctx.TellSuccess(out)
}

func (n *VolcTlsSearchLogsNode) Destroy() {
	n.cfg = volcTlsSearchLogsConfig{}
	n.defaultQueryTmpl = nil
}

func init() {
	_ = rulego.Registry.Register(&VolcTlsSearchLogsNode{})
}
