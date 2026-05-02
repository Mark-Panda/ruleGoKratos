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
	"strings"
	"time"

	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/base"
	"github.com/rulego/rulego/utils/el"
	"github.com/rulego/rulego/utils/maps"
)

func init() {
	_ = rulego.Registry.Register(&SourcegraphTokenVerifyComponent{})
}

// SourcegraphTokenVerifyComponent 校验 SourceGraph API Token 是否有效，
// 通过调用 currentUser GraphQL 查询判断 token 有效性。
type SourcegraphTokenVerifyComponent struct {
	Config SourcegraphTokenVerifyConfiguration

	endpointTpl    el.Template
	accessTokenTpl el.Template
}

type SourcegraphTokenVerifyConfiguration struct {
	// Sourcegraph 服务地址，例如 "https://sourcegraph.xxxx.tv"
	Endpoint string `json:"endpoint"`

	// Sourcegraph API 访问令牌
	AccessToken string `json:"accessToken"`

	// HTTP 请求超时秒数，默认 15
	TimeoutSec int `json:"timeoutSec"`
}

func (c *SourcegraphTokenVerifyComponent) New() types.Node {
	return &SourcegraphTokenVerifyComponent{Config: SourcegraphTokenVerifyConfiguration{
		TimeoutSec: 15,
	}}
}

func (c *SourcegraphTokenVerifyComponent) Type() string {
	return "x/sourcegraphTokenVerify"
}

func (c *SourcegraphTokenVerifyComponent) Def() types.ComponentForm {
	return types.ComponentForm{
		Label: "sourcegraphTokenVerify",
		Desc:  "校验 SourceGraph API Token 有效性；有效走 Success，无效走 Failure",
	}
}

func (c *SourcegraphTokenVerifyComponent) Init(_ types.Config, configuration types.Configuration) error {
	if err := maps.Map2Struct(configuration, &c.Config); err != nil {
		return err
	}
	if c.Config.TimeoutSec <= 0 {
		c.Config.TimeoutSec = 15
	}

	endpointTpl, err := el.NewTemplate(strings.TrimSpace(c.Config.Endpoint))
	if err != nil {
		return err
	}
	c.endpointTpl = endpointTpl

	accessTokenTpl, err := el.NewTemplate(strings.TrimSpace(c.Config.AccessToken))
	if err != nil {
		return err
	}
	c.accessTokenTpl = accessTokenTpl

	return nil
}

func (c *SourcegraphTokenVerifyComponent) OnMsg(ctx types.RuleContext, msg types.RuleMsg) {
	env := base.NodeUtils.GetEvnAndMetadata(ctx, msg)

	endpoint := strings.TrimRight(strings.TrimSpace(c.endpointTpl.ExecuteAsString(env)), "/")
	if endpoint == "" {
		ctx.TellFailure(msg, errors.New("sourcegraphTokenVerify: endpoint 为空"))
		return
	}

	accessToken := strings.TrimSpace(c.accessTokenTpl.ExecuteAsString(env))
	if accessToken == "" {
		ctx.TellFailure(msg, errors.New("sourcegraphTokenVerify: accessToken 为空"))
		return
	}

	gqlURL, err := url.JoinPath(endpoint, ".api", "graphql")
	if err != nil {
		ctx.TellFailure(msg, fmt.Errorf("sourcegraphTokenVerify: 拼接 GraphQL URL 失败: %w", err))
		return
	}

	username, err := verifySourcegraphToken(gqlURL, accessToken, c.Config.TimeoutSec)
	if err != nil {
		ctx.TellFailure(msg, fmt.Errorf("sourcegraphTokenVerify: token 校验失败: %w", err))
		return
	}

	out := msg.Copy()
	if out.Metadata == nil {
		out.Metadata = types.NewMetadata()
	}
	out.Metadata.PutValue("sourcegraph_username", username)
	out.Metadata.PutValue("sourcegraph_endpoint", endpoint)

	payload, _ := json.Marshal(map[string]interface{}{
		"valid":    true,
		"username": username,
		"endpoint": endpoint,
	})
	out.SetData(string(payload))

	ctx.TellSuccess(out)
}

func (c *SourcegraphTokenVerifyComponent) Destroy() {
	c.Config = SourcegraphTokenVerifyConfiguration{}
	c.endpointTpl = nil
	c.accessTokenTpl = nil
}

func (c *SourcegraphTokenVerifyComponent) Close() error { return nil }

const sourcegraphCurrentUserQuery = `query { currentUser { username } }`

type sourcegraphCurrentUserResponse struct {
	Data struct {
		CurrentUser *struct {
			Username string `json:"username"`
		} `json:"currentUser"`
	} `json:"data"`
	Errors []json.RawMessage `json:"errors"`
}

func verifySourcegraphToken(gqlURL, accessToken string, timeoutSec int) (string, error) {
	payload := map[string]interface{}{
		"query": sourcegraphCurrentUserQuery,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	ctxHTTP, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctxHTTP, http.MethodPost, gqlURL, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "token "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncateForTracerLog(strings.TrimSpace(string(respBody)), 512))
	}

	var gqlResp sourcegraphCurrentUserResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		preview := truncateForTracerLog(strings.TrimSpace(string(respBody)), 320)
		return "", fmt.Errorf("响应非 JSON，正文片段: %q: %w", preview, err)
	}

	if len(gqlResp.Errors) > 0 {
		return "", fmt.Errorf("GraphQL 错误: %s", string(gqlResp.Errors[0]))
	}

	if gqlResp.Data.CurrentUser == nil || strings.TrimSpace(gqlResp.Data.CurrentUser.Username) == "" {
		return "", errors.New("currentUser 为空，token 可能无效或已过期")
	}

	return gqlResp.Data.CurrentUser.Username, nil
}
