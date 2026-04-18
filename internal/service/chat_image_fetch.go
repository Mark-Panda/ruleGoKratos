package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	v1 "ruleGoKratos/api/rulego/v1"
	"ruleGoKratos/internal/biz"
)

const (
	chatImageFetchMaxBytes   = 6 * 1024 * 1024
	chatImageFetchMaxURLs    = 4
	chatImageFetchTimeout    = 20 * time.Second
	chatImageFetchUserAgent = "RuleGoKratos-Chat/1.0"
)

// 从正文中提取 https 直链，由服务端拉取为附件，使多模态模型能「看到」图/视频，而无需模型假装访问外网。

var reChatHTTPSURL = regexp.MustCompile(`https://[^\s<>"'()，。、\]]+`)

func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() {
		return true
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	// IPv6 ULA fc00::/7
	if len(ip) == net.IPv6len && (ip[0]&0xfe) == 0xfc {
		return true
	}
	return false
}

func assertHostResolvesToPublicIP(ctx context.Context, host string) error {
	if host == "" {
		return errors.New("empty host")
	}
	hst, _, err := net.SplitHostPort(host)
	if err == nil {
		host = hst
	}
	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return fmt.Errorf("地址不可访问: %s", ip)
		}
		return nil
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return err
	}
	if len(ips) == 0 {
		return errors.New("无法解析主机名")
	}
	for _, ia := range ips {
		if isBlockedIP(ia.IP) {
			return fmt.Errorf("地址不可访问: %s", ia.IP)
		}
	}
	return nil
}

func trimURLTail(s string) string {
	return strings.TrimRight(s, ".,;:!?)）】。、，；：！？【】]'\"")
}

func collectMessageHTTPSURLs(msg string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, m := range reChatHTTPSURL.FindAllString(msg, -1) {
		u := trimURLTail(m)
		if u == "" {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, u)
		if len(out) >= chatImageFetchMaxURLs {
			break
		}
	}
	return out
}

func filenameForImageURL(u *url.URL) string {
	if p := path.Base(u.Path); p != "" && p != "/" {
		return p
	}
	return "fetched_from_url"
}

// fetchImageFromHTTPS 仅支持 https，且仅当魔数为图/视频时返回；否则 (nil, false, nil)。
func fetchImageFromHTTPS(ctx context.Context, raw string) (*v1.ChatAttachment, bool, error) {
	raw = strings.TrimSpace(raw)
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return nil, false, nil
	}
	if err := assertHostResolvesToPublicIP(ctx, u.Host); err != nil {
		return nil, false, err
	}

	ctx, cancel := context.WithTimeout(ctx, chatImageFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", chatImageFetchUserAgent)
	req.Header.Set("Accept", "image/*,video/*,*/*")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, chatImageFetchMaxBytes+1))
	if err != nil {
		return nil, false, err
	}
	if len(body) > chatImageFetchMaxBytes {
		return nil, false, errors.New("响应过大")
	}

	sniff := biz.SniffBinaryMIME(body)
	if sniff == "" {
		return nil, false, nil
	}
	if !strings.HasPrefix(sniff, "image/") && !strings.HasPrefix(sniff, "video/") {
		return nil, false, nil
	}

	declared := ""
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		declared, _, _ = mime.ParseMediaType(ct)
	}
	mimeType := sniff
	if strings.HasPrefix(declared, "image/") || strings.HasPrefix(declared, "video/") {
		mimeType = declared
	}

	b64 := base64.StdEncoding.EncodeToString(body)
	return &v1.ChatAttachment{
		Filename:      filenameForImageURL(u),
		MimeType:      mimeType,
		ContentBase64: b64,
	}, true, nil
}

// enrichReqFromMessageImageURLs 将 message 中出现的 https 图/视频链接拉取后追加到 attachments（失败则静默跳过）。
func enrichReqFromMessageImageURLs(ctx context.Context, req *v1.ChatStreamReq) {
	if req == nil {
		return
	}
	msg := req.GetMessage()
	if strings.TrimSpace(msg) == "" {
		return
	}
	urls := collectMessageHTTPSURLs(msg)
	if len(urls) == 0 {
		return
	}
	for _, raw := range urls {
		att, ok, err := fetchImageFromHTTPS(ctx, raw)
		if err != nil || !ok || att == nil {
			continue
		}
		req.Attachments = append(req.Attachments, att)
	}
}
