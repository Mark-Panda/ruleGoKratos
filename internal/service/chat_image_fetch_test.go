package service

import (
	"net"
	"testing"
)

func TestCollectMessageHTTPSURLs_dedupeAndLimit(t *testing.T) {
	msg := `请看 https://a.com/x.png 与 https://b.com/y.jpg，重复 https://a.com/x.png`
	urls := collectMessageHTTPSURLs(msg)
	if len(urls) != 2 {
		t.Fatalf("got %v", urls)
	}
	if urls[0] != "https://a.com/x.png" || urls[1] != "https://b.com/y.jpg" {
		t.Fatalf("got %v", urls)
	}
}

func TestTrimURLTail(t *testing.T) {
	if trimURLTail("https://x.com/a.png)。") != "https://x.com/a.png" {
		t.Fatal()
	}
}

func TestIsBlockedIP(t *testing.T) {
	if !isBlockedIP(netParseIPMust("127.0.0.1")) {
		t.Fatal()
	}
	if isBlockedIP(netParseIPMust("8.8.8.8")) {
		t.Fatal()
	}
}

func netParseIPMust(s string) net.IP {
	ip := net.ParseIP(s)
	if ip == nil {
		panic(s)
	}
	return ip
}
