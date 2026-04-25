package data

import (
	"encoding/json"
	"fmt"
	"strings"
)

func parseJSONStringArray(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func parseJSONStringMap(raw string) (map[string]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, err
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = fmt.Sprint(v)
	}
	return out, nil
}

func headersMapFromJSON(raw string) map[string]string {
	m, err := parseJSONStringMap(raw)
	if err != nil {
		return nil
	}
	return m
}
