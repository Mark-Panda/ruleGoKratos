package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/base"
	"github.com/rulego/rulego/utils/el"
	"github.com/rulego/rulego/utils/maps"
)

func init() {
	_ = rulego.Registry.Register(&SourcegraphTokenCreateComponent{})
}

var (
	playwrightOnce       sync.Once
	playwrightInstallErr error
)

// SourcegraphTokenCreateComponent 通过 LDAP 凭证走 GitLab OAuth 流程登录 Sourcegraph，
// 然后在 Token 设置页创建 Access Token。使用 playwright-go 实现，无需已有 Token。
type SourcegraphTokenCreateComponent struct {
	Config SourcegraphTokenCreateConfiguration

	endpointTpl     el.Template
	noteTpl         el.Template
	expiresAtTpl    el.Template
	scopeTpl        el.Template
	ldapUsernameTpl el.Template
	ldapPasswordTpl el.Template
	gitlabHostTpl   el.Template
	headlessTpl     el.Template
}

type SourcegraphTokenCreateConfiguration struct {
	// Sourcegraph 服务地址，例如 "https://sourcegraph.xxxxx.tv"
	Endpoint string `json:"endpoint"`

	// Token 名称/备注，默认 "cli-token"
	Note string `json:"note"`

	// ISO 8601 过期时间，例如 "2029-05-01T00:00:00Z"；空则默认 3 年后
	ExpiresAt string `json:"expiresAt"`

	// Token 权限范围，空默认 USER；可选 "USER"、"SITE_ADMIN"
	Scope string `json:"scope"`

	// LDAP 用户名，必填
	LdapUsername string `json:"ldapUsername"`

	// LDAP 密码，必填
	LdapPassword string `json:"ldapPassword"`

	// GitLab 主机地址，默认 "gitlab.xxxx.tv"
	GitlabHost string `json:"gitlabHost"`

	// Playwright headless 模式，默认 "true"
	Headless string `json:"headless"`

	// 页面操作超时毫秒数，默认 60000
	TimeoutMs int `json:"timeoutMs"`
}

func (c *SourcegraphTokenCreateComponent) New() types.Node {
	return &SourcegraphTokenCreateComponent{Config: SourcegraphTokenCreateConfiguration{
		Note:       "cli-token",
		GitlabHost: "gitlab.xxx.tv",
		Headless:   "true",
		TimeoutMs:  60000,
	}}
}

func (c *SourcegraphTokenCreateComponent) Type() string {
	return "x/sourcegraphTokenCreate"
}

func (c *SourcegraphTokenCreateComponent) Def() types.ComponentForm {
	return types.ComponentForm{
		Label: "sourcegraphTokenCreate",
		Desc:  "通过 LDAP 凭证登录 Sourcegraph 并创建 Access Token；无需已有 Token",
	}
}

func (c *SourcegraphTokenCreateComponent) Init(_ types.Config, configuration types.Configuration) error {
	if err := maps.Map2Struct(configuration, &c.Config); err != nil {
		return err
	}
	if c.Config.TimeoutMs <= 0 {
		c.Config.TimeoutMs = 60000
	}
	if strings.TrimSpace(c.Config.GitlabHost) == "" {
		c.Config.GitlabHost = "gitlab.xxx.tv"
	}
	if strings.TrimSpace(c.Config.Headless) == "" {
		c.Config.Headless = "true"
	}

	tpls := map[*el.Template]string{
		&c.endpointTpl:     c.Config.Endpoint,
		&c.noteTpl:         c.Config.Note,
		&c.expiresAtTpl:    c.Config.ExpiresAt,
		&c.scopeTpl:        c.Config.Scope,
		&c.ldapUsernameTpl: c.Config.LdapUsername,
		&c.ldapPasswordTpl: c.Config.LdapPassword,
		&c.gitlabHostTpl:   c.Config.GitlabHost,
		&c.headlessTpl:     c.Config.Headless,
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

func (c *SourcegraphTokenCreateComponent) OnMsg(ctx types.RuleContext, msg types.RuleMsg) {
	env := base.NodeUtils.GetEvnAndMetadata(ctx, msg)

	endpoint := strings.TrimRight(strings.TrimSpace(c.endpointTpl.ExecuteAsString(env)), "/")
	if endpoint == "" {
		ctx.TellFailure(msg, errors.New("sourcegraphTokenCreate: endpoint 为空"))
		return
	}

	ldapUsername := strings.TrimSpace(c.ldapUsernameTpl.ExecuteAsString(env))
	ldapPassword := strings.TrimSpace(c.ldapPasswordTpl.ExecuteAsString(env))
	if ldapUsername == "" || ldapPassword == "" {
		ctx.TellFailure(msg, errors.New("sourcegraphTokenCreate: ldapUsername 和 ldapPassword 不能为空"))
		return
	}

	gitlabHost := strings.TrimSpace(c.gitlabHostTpl.ExecuteAsString(env))
	if gitlabHost == "" {
		gitlabHost = "gitlab.xxxx.tv"
	}
	note := strings.TrimSpace(c.noteTpl.ExecuteAsString(env))
	if note == "" {
		note = "cli-token"
	}
	expiresAt := strings.TrimSpace(c.expiresAtTpl.ExecuteAsString(env))
	if expiresAt == "" {
		expiresAt = defaultExpiry()
	}
	headless := parseTruthy(c.headlessTpl.ExecuteAsString(env))
	timeoutMs := c.Config.TimeoutMs

	result, err := createTokenViaPlaywright(endpoint, ldapUsername, ldapPassword, gitlabHost, note, expiresAt, headless, timeoutMs)
	if err != nil {
		ctx.TellFailure(msg, fmt.Errorf("sourcegraphTokenCreate: %w", err))
		return
	}

	out := msg.Copy()
	if out.Metadata == nil {
		out.Metadata = types.NewMetadata()
	}
	mergeTraceMetadata(msg, out)

	out.Metadata.PutValue("sourcegraph_token", result.Token)
	out.Metadata.PutValue("sourcegraph_token_note", result.Note)
	out.Metadata.PutValue("sourcegraph_endpoint", endpoint)

	payload, _ := json.Marshal(map[string]interface{}{
		"token":    result.Token,
		"note":     result.Note,
		"endpoint": endpoint,
	})
	out.SetData(string(payload))

	ctx.TellSuccess(out)
}

func (c *SourcegraphTokenCreateComponent) Destroy() {
	c.Config = SourcegraphTokenCreateConfiguration{}
	c.endpointTpl = nil
	c.noteTpl = nil
	c.expiresAtTpl = nil
	c.scopeTpl = nil
	c.ldapUsernameTpl = nil
	c.ldapPasswordTpl = nil
	c.gitlabHostTpl = nil
	c.headlessTpl = nil
}

func (c *SourcegraphTokenCreateComponent) Close() error { return nil }

// ---------------------------------------------------------------------------
// Playwright 流程：忠实移植 sourcegraph-token-playwright.js
// ---------------------------------------------------------------------------

type tokenResult struct {
	Token string
	Note  string
}

func createTokenViaPlaywright(endpoint, username, password, gitlabHost, note, expiresAt string, headless bool, timeoutMs int) (*tokenResult, error) {
	gitlabLDAPRe := regexp.MustCompile(
		regexp.QuoteMeta(gitlabHost) + `/(users/sign_in|users/auth/ldap)`,
	)
	timeout := float64(timeoutMs)

	// 首次运行时自动安装 Playwright driver（仅执行一次）
	// 使用 SkipInstallBrowsers：Docker 中已预装 Chromium，无需运行时再下载
	playwrightOnce.Do(func() {
		playwrightInstallErr = playwright.Install(&playwright.RunOptions{
			SkipInstallBrowsers: true,
		})
	})
	if playwrightInstallErr != nil {
		return nil, fmt.Errorf("安装 Playwright 失败: %w（Docker 中请预装 chromium，参考 Dockerfile.all）", playwrightInstallErr)
	}

	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("启动 Playwright 失败: %w", err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(headless),
		Args: []string{
			"--no-sandbox",
			"--disable-setuid-sandbox",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("启动浏览器失败: %w", err)
	}
	defer browser.Close()

	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		Viewport: &playwright.Size{Width: 1280, Height: 720},
	})
	if err != nil {
		return nil, fmt.Errorf("创建浏览器上下文失败: %w", err)
	}
	defer context.Close()

	page, err := context.NewPage()
	if err != nil {
		return nil, fmt.Errorf("创建页面失败: %w", err)
	}

	// [1/6] 打开 Sourcegraph
	if _, err := page.Goto(endpoint, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(timeout),
	}); err != nil {
		return nil, fmt.Errorf("打开 Sourcegraph 失败: %w", err)
	}

	// [2/6] 选择 GitLab 登录方式
	gitLabLocator := page.Locator("button:has-text(\"GitLab\"), a:has-text(\"GitLab\")").First()
	if err := gitLabLocator.Click(playwright.LocatorClickOptions{
		Timeout: playwright.Float(10000),
	}); err != nil {
		return nil, fmt.Errorf("点击 GitLab 登录按钮失败: %w", err)
	}

	// [3/6] 等待 GitLab 登录页
	if err := page.WaitForURL(gitlabLDAPRe, playwright.PageWaitForURLOptions{
		Timeout: playwright.Float(30000),
	}); err != nil {
		return nil, fmt.Errorf("等待 GitLab 登录页失败: %w", err)
	}

	// [4/6] 填写 LDAP 凭证
	usernameSel := "#username, [name=\"username\"], input[type=\"text\"]:visible"
	if err := waitAndType(page, usernameSel, username, timeout); err != nil {
		return nil, fmt.Errorf("填写用户名失败: %w", err)
	}
	passwordSel := "#password, [name=\"password\"], input[type=\"password\"]:visible"
	if err := waitAndType(page, passwordSel, password, timeout); err != nil {
		return nil, fmt.Errorf("填写密码失败: %w", err)
	}
	//lint:ignore SA1019 "deprecated playwright API"
	if err := page.Click("button[type=\"submit\"], input[type=\"submit\"]"); err != nil {
		return nil, fmt.Errorf("点击登录按钮失败: %w", err)
	}

	// [5/6] 处理 OAuth 回调
	sgGlob := endpoint + "/-*"
	_ = page.WaitForURL(sgGlob, playwright.PageWaitForURLOptions{
		Timeout: playwright.Float(30000),
	})

	authBtn := page.Locator("button:has-text(\"Authorize\"), input[value=\"Authorize\"]").First()
	visible, _ := authBtn.IsVisible(playwright.LocatorIsVisibleOptions{
		Timeout: playwright.Float(5000),
	})
	if visible {
		if err := authBtn.Click(); err != nil {
			return nil, fmt.Errorf("点击 Authorize 按钮失败: %w", err)
		}
		_ = page.WaitForURL(sgGlob, playwright.PageWaitForURLOptions{
			Timeout: playwright.Float(30000),
		})
	}

	// [6/6] 创建 Access Token
	tokenPageURL := endpoint + "/user/settings/tokens/new"
	if _, err := page.Goto(tokenPageURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(timeout),
	}); err != nil {
		return nil, fmt.Errorf("打开 Token 创建页失败: %w", err)
	}

	currentURL := page.URL()
	if strings.Contains(currentURL, "/search") || strings.HasSuffix(currentURL, "/") {
		adminURL := endpoint + "/-/admin/tokens"
		if _, err := page.Goto(adminURL, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateNetworkidle,
			Timeout:   playwright.Float(timeout),
		}); err != nil {
			return nil, fmt.Errorf("打开管理员 Token 页失败: %w", err)
		}
	}

	// 若已在 /new 页（表单直接可见），跳过 "Create access token" 按钮
	createBtn := page.Locator("button:has-text(\"Create access token\"), button:has-text(\"Generate token\")")
	createBtnVisible, _ := createBtn.First().IsVisible(playwright.LocatorIsVisibleOptions{
		Timeout: playwright.Float(3000),
	})
	if createBtnVisible {
		_ = createBtn.First().Click(playwright.LocatorClickOptions{
			Timeout: playwright.Float(10000),
		})
	}

	tokenNameSel := "input[name=\"tokenName\"], input[placeholder*=\"token\" i], input[id*=\"name\"]"
	if err := waitAndType(page, tokenNameSel, note, timeout); err != nil {
		return nil, fmt.Errorf("填写 Token 名称失败: %w", err)
	}

	expiresInput := page.Locator("input[type=\"date\"], input[name*=\"expir\" i], input[id*=\"expir\" i]").First()
	expiresVisible, _ := expiresInput.IsVisible(playwright.LocatorIsVisibleOptions{
		Timeout: playwright.Float(5000),
	})
	if expiresVisible {
		_ = expiresInput.Click(playwright.LocatorClickOptions{ClickCount: playwright.Int(3)})
		_ = expiresInput.Type(expiresAt) //lint:ignore SA1019 "deprecated playwright API"
	}

	submitBtn := page.Locator("button:has-text(\"Create token\"), button:has-text(\"Generate token\"), button[type=\"submit\"]").First()
	if err := submitBtn.Click(); err != nil {
		return nil, fmt.Errorf("点击创建 Token 按钮失败: %w", err)
	}

	// 等待 Token 值出现
	if _, err := page.WaitForSelector( //lint:ignore SA1019 "deprecated playwright API"
		".access-token-value, [data-testid=\"token-value\"], .token-value, input[readonly][value]",
		playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(15000),
		},
	); err != nil {
		return nil, fmt.Errorf("等待 Token 值出现失败: %w", err)
	}

	token, err := extractToken(page)
	if err != nil || token == "" {
		return nil, fmt.Errorf("无法提取 Token: %w", err)
	}

	return &tokenResult{Token: token, Note: note}, nil
}

// waitAndType 等待元素可见后清空并输入文本，复刻 JS 脚本的 waitAndType。
func waitAndType(page playwright.Page, selector, text string, timeout float64) error {
	el, err := page.WaitForSelector(selector, playwright.PageWaitForSelectorOptions{ //lint:ignore SA1019 "deprecated playwright API"
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(timeout),
	})
	if err != nil {
		return err
	}
	_ = el.Click(playwright.ElementHandleClickOptions{ClickCount: playwright.Int(3)}) //lint:ignore SA1019 "deprecated playwright API"
	err = el.Type(text) //lint:ignore SA1019 "deprecated playwright API"
	return err
}

// extractToken 从页面中提取新创建的 Token 值，复刻 JS 脚本的 extractToken。
func extractToken(page playwright.Page) (string, error) {
	selectors := []string{
		".access-token-value",
		"[data-testid=\"token-value\"]",
		".token-value",
		"input[readonly][value]",
		"input[type=\"text\"][readonly]",
	}
	tokenRe := regexp.MustCompile(`^[a-f0-9]{20,}$`)
	for _, sel := range selectors {
		loc := page.Locator(sel).First()
		visible, _ := loc.IsVisible(playwright.LocatorIsVisibleOptions{
			Timeout: playwright.Float(2000),
		})
		if !visible {
			continue
		}
		// 先尝试 inputValue
		val, err := loc.InputValue()
		if err == nil && tokenRe.MatchString(val) {
			return val, nil
		}
		// 再尝试 textContent
		val, err = loc.TextContent()
		if err == nil && tokenRe.MatchString(strings.TrimSpace(val)) {
			return strings.TrimSpace(val), nil
		}
	}
	return "", errors.New("未找到 Token 值")
}

// ---------------------------------------------------------------------------
// 工具函数
// ---------------------------------------------------------------------------

func defaultExpiry() string {
	d := time.Now().AddDate(3, 0, 0)
	return d.Format("2006-01-02")
}

func parseTruthy(s string) bool {
	v := strings.ToLower(strings.TrimSpace(s))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}
