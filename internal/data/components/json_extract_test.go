package data

import (
	"strings"
	"testing"
)

func TestNormalizeTopLevelArrayToStringKeyMap(t *testing.T) {
	in := []interface{}{
		map[string]interface{}{"name": "a"},
		map[string]interface{}{"name": "b"},
	}
	got := normalizeTopLevelArrayToStringKeyMap(in)
	obj, ok := got.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map output, got: %#v", got)
	}
	if len(obj) != 2 {
		t.Fatalf("expected 2 keys, got: %#v", obj)
	}
	if _, exists := obj["0"]; !exists {
		t.Fatalf("expected string key '0', got: %#v", obj)
	}
	if _, exists := obj["1"]; !exists {
		t.Fatalf("expected string key '1', got: %#v", obj)
	}
}

func TestParseJsonWithFixes_StrictExtractFromWrappedText(t *testing.T) {
	input := `前缀说明
{"data":[{"name":"svc-alpha","spaceName":"ns-alpha"}]}
后缀`
	got := parseJsonWithFixes(input, "")
	if !got.Success {
		t.Fatalf("expected success, got error: %s", got.Error)
	}
	obj, ok := got.ExtractedJson.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object json, got: %#v", got.ExtractedJson)
	}
	if _, ok := obj["data"].([]interface{}); !ok {
		t.Fatalf("expected data array, got: %#v", obj["data"])
	}
}

func TestParseJsonWithFixes_StrictExtractFromMarkdownFence(t *testing.T) {
	input := "说明文字\n```json\n{\"ok\":true,\"value\":1}\n```\n后续文字"
	got := parseJsonWithFixes(input, "md")
	if !got.Success {
		t.Fatalf("expected success, got error: %s", got.Error)
	}
	obj, ok := got.ExtractedJson.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object json, got: %#v", got.ExtractedJson)
	}
	if v, ok := obj["ok"].(bool); !ok || !v {
		t.Fatalf("expected ok=true, got: %#v", obj["ok"])
	}
}

func TestParseJsonWithFixes_StrictUnwrapJSONString(t *testing.T) {
	input := `"{\"data\":[{\"name\":\"svc-a\",\"framework\":\"kratos\"}]}"`
	got := parseJsonWithFixes(input, "aggressive")
	if !got.Success {
		t.Fatalf("expected success, got error: %s", got.Error)
	}
	obj, ok := got.ExtractedJson.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object json, got: %#v", got.ExtractedJson)
	}
	if _, ok := obj["data"].([]interface{}); !ok {
		t.Fatalf("expected data array, got: %#v", obj["data"])
	}
}

func TestParseJsonWithFixes_RepairOnlyInputStillRecovered(t *testing.T) {
	input := `{'name':'demo','n':1,}`
	got := parseJsonWithFixes(input, "aggressive")
	if !got.Success {
		t.Fatalf("expected recovered success, got error: %s", got.Error)
	}
	if len(got.RepairStrategies) == 0 {
		t.Fatalf("expected non-empty repair strategies, got: %#v", got.RepairStrategies)
	}
}

func TestParseJsonWithFixes_StrictExtractFromEmbeddedMarkdownJSON(t *testing.T) {
	outer := "{\"payload\":\"说明\\n```json\\n{\\\"data\\\":[{\\\"name\\\":\\\"svc-alpha\\\",\\\"spaceName\\\":\\\"ns-alpha\\\"}]}\\n```\",\"type\":\"trace\"}"
	got := parseJsonWithFixes(outer, "")
	if !got.Success {
		t.Fatalf("expected success, got error: %s", got.Error)
	}
	obj, ok := got.ExtractedJson.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object json, got: %#v", got.ExtractedJson)
	}
	if _, ok := obj["data"].([]interface{}); !ok {
		t.Fatalf("expected embedded data array selected, got: %#v", obj)
	}
}

func TestParseJsonWithFixes_PrimitiveFallback(t *testing.T) {
	got := parseJsonWithFixes("123", "")
	if !got.Success {
		t.Fatalf("expected primitive fallback success, got error: %s", got.Error)
	}
	v, ok := got.ExtractedJson.(float64)
	if !ok || v != 123 {
		t.Fatalf("expected primitive number 123, got: %#v", got.ExtractedJson)
	}
	if len(got.RepairStrategies) == 0 || got.RepairStrategies[0] != "primitive_fallback" {
		t.Fatalf("expected primitive_fallback strategy, got: %#v", got.RepairStrategies)
	}
}

func TestExtractBalancedJSONFragments_PrefersLongestAndDeduplicates(t *testing.T) {
	input := strings.Join([]string{
		"prefix",
		`{"a":1}`,
		`{"a":1}`,
		`{"data":[{"name":"svc-a"},{"name":"svc-b"}]}`,
		"suffix",
	}, " ")
	frags := extractBalancedJSONFragments(input, 2)
	if len(frags) != 2 {
		t.Fatalf("expected 2 fragments, got: %#v", frags)
	}
	if !strings.Contains(frags[0], `"data":[`) {
		t.Fatalf("expected longest fragment first, got: %#v", frags)
	}
	if frags[0] == frags[1] {
		t.Fatalf("expected deduplicated fragments, got: %#v", frags)
	}
}

func TestCutBalancedJSONFragment_HandlesEscapedQuotes(t *testing.T) {
	frag, ok := cutBalancedJSONFragment(`{"text":"a\"b","arr":[1,2]} trailing`)
	if !ok {
		t.Fatal("expected balanced fragment found")
	}
	if frag != `{"text":"a\"b","arr":[1,2]}` {
		t.Fatalf("unexpected fragment: %s", frag)
	}
}

func TestCutBalancedJSONFragment_DetectsInvalidCloseOrder(t *testing.T) {
	_, ok := cutBalancedJSONFragment(`{"a":[1,2}`)
	if ok {
		t.Fatal("expected invalid close order to fail")
	}
}

func TestExtractEmbeddedTextCandidatesFromJSON_SelectsUsefulTexts(t *testing.T) {
	text := "{\"meta\":\"ok\",\"payload\":\"```json\\n{\\\"ok\\\":true}\\n```\",\"nested\":{\"line\":\"prefix {\\\"a\\\":1} suffix\"},\"tiny\":\"x\"}"
	candidates := extractEmbeddedTextCandidatesFromJSON(text, 3)
	if len(candidates) == 0 {
		t.Fatal("expected non-empty embedded candidates")
	}
	foundMarkdown := false
	for _, c := range candidates {
		if strings.Contains(c, "```json") {
			foundMarkdown = true
			break
		}
	}
	if !foundMarkdown {
		t.Fatalf("expected markdown candidate included, got: %#v", candidates)
	}
}

func TestExtractJsonHelpers_MarkdownTaggedAssignment(t *testing.T) {
	md := extractJsonFromMarkdown("a\n```json\n{\"ok\":true}\n```\nb")
	if md != `{"ok":true}` {
		t.Fatalf("unexpected markdown extraction: %q", md)
	}

	tagged := extractJsonFromTaggedBlock(`<result>{"count":2}</result>`)
	if tagged != `{"count":2}` {
		t.Fatalf("unexpected tagged extraction: %q", tagged)
	}

	assign := extractJsonFromAssignment(`const result = {"name":"demo","n":1};`)
	if assign != `{"name":"demo","n":1}` {
		t.Fatalf("unexpected assignment extraction: %q", assign)
	}
}

func TestSelectBestResult_TieBreakers(t *testing.T) {
	r1 := Result{CandidateScore: 200, SchemaMissing: []string{"a", "b"}, RepairStrategies: []string{"x"}}
	r2 := Result{CandidateScore: 200, SchemaMissing: []string{"a"}, RepairStrategies: []string{"x", "y"}}
	best := selectBestResult([]Result{r1, r2})
	if len(best.SchemaMissing) != 1 {
		t.Fatalf("expected fewer schema missing preferred, got: %#v", best)
	}

	r3 := Result{CandidateScore: 180, SchemaMissing: []string{"a"}, RepairStrategies: []string{"x", "y"}}
	r4 := Result{CandidateScore: 180, SchemaMissing: []string{"a"}, RepairStrategies: []string{"x"}}
	best2 := selectBestResult([]Result{r3, r4})
	if len(best2.RepairStrategies) != 1 {
		t.Fatalf("expected fewer repairs preferred, got: %#v", best2)
	}
}

func TestApplySchemaCompletion_ComplexNestedPaths(t *testing.T) {
	parsed := map[string]interface{}{
		"data": []interface{}{
			map[string]interface{}{"name": "svc-a"},
		},
	}
	compiled := compileSchemaPaths([]string{"data[].name", "data[].spaceName", "meta.total"})
	out, matched, missing, changed := applySchemaCompletion(parsed, compiled)
	if !changed {
		t.Fatalf("expected schema completion changed output, got out=%#v", out)
	}
	if matched != 1 {
		t.Fatalf("expected 1 matched schema, got %d", matched)
	}
	if len(missing) != 2 {
		t.Fatalf("expected 2 missing schemas, got %#v", missing)
	}
	obj := out.(map[string]interface{})
	meta, ok := obj["meta"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected meta object created, got: %#v", obj["meta"])
	}
	if _, exists := meta["total"]; !exists {
		t.Fatalf("expected meta.total created, got: %#v", meta)
	}
	rows := obj["data"].([]interface{})
	first := rows[0].(map[string]interface{})
	if _, exists := first["spaceName"]; !exists {
		t.Fatalf("expected spaceName ensured, got: %#v", first)
	}
}

func TestBuildRepairReport_ContainsCoreFields(t *testing.T) {
	report := buildRepairReport(Result{
		Success:              true,
		SourceStrategy:       "balanced_fragment",
		RepairStrategies:     []string{"unwrap_json_string"},
		CandidateScore:       123,
		TruncatedTailDropped: false,
		ExtractedJson: map[string]interface{}{
			"ok": true,
		},
	})
	m, ok := report.(map[string]interface{})
	if !ok {
		t.Fatalf("expected report map, got: %#v", report)
	}
	if m["source_strategy"] != "balanced_fragment" {
		t.Fatalf("unexpected source strategy: %#v", m["source_strategy"])
	}
	if m["score"] != 123 {
		t.Fatalf("unexpected score: %#v", m["score"])
	}
}

func TestParseJsonWithFixes_TruncatedLargeListStillExtractsCoreData(t *testing.T) {
	input := "已从示例数据源提取后端应用信息，共 **128个** 服务：\n```json\n" + `{
  "data": [
    {"name":"svc-alpha","spaceName":"ns-alpha","manager":"owner-a","language":"Go","framework":"kratos"},
    {"name":"svc-beta-consumer","spaceName":"ns-beta","manager":"owner-b","language":"Go","framework":"gin"},
    {"name":"svc-gamma","spaceName":"ns-beta","manager":"` + "\n```"

	got := parseJsonWithFixes(input, "")
	if !got.Success {
		t.Fatalf("expected success for truncated large list, got error: %s", got.Error)
	}
	obj, ok := got.ExtractedJson.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object json, got: %#v", got.ExtractedJson)
	}
	rows, ok := obj["data"].([]interface{})
	if !ok || len(rows) < 2 {
		t.Fatalf("expected at least 2 rows, got: %#v", obj["data"])
	}
	names := make(map[string]bool, len(rows))
	for _, row := range rows {
		m, ok := row.(map[string]interface{})
		if !ok {
			continue
		}
		if n, ok := m["name"].(string); ok {
			names[n] = true
		}
	}
	if !names["svc-alpha"] || !names["svc-beta-consumer"] {
		t.Fatalf("expected core rows retained, got names: %#v", names)
	}
}

func TestParseJsonWithFixes_WrappedMessageDataFieldComplexScene(t *testing.T) {
	innerJSON := `{"data":[{"name":"svc-alpha","spaceName":"ns-alpha"},{"name":"svc-delta-consumer","spaceName":"ns-beta"}]}`
	outer := `{"data":{"data":"` + strings.ReplaceAll(strings.ReplaceAll(innerJSON, `\`, `\\`), `"`, `\"`) + `"},"type":"111"}`

	got := parseJsonWithFixes(outer, "")
	if !got.Success {
		t.Fatalf("expected success for wrapped complex scene, got error: %s", got.Error)
	}
	obj, ok := got.ExtractedJson.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object json, got: %#v", got.ExtractedJson)
	}
	rows, ok := obj["data"].([]interface{})
	if !ok || len(rows) < 2 {
		t.Fatalf("expected parsed embedded data array, got: %#v", obj)
	}
}

func TestParseJsonWithFixes_TableDrivenComplexRegression(t *testing.T) {
	type rowAssertion struct {
		minRows int
		names   []string
	}
	type testCase struct {
		name             string
		input            string
		expectSuccess    bool
		expectStrategyIn string
		rowAssert        *rowAssertion
	}

	innerJSON := `{"data":[{"name":"svc-alpha","spaceName":"ns-alpha"},{"name":"svc-delta-consumer","spaceName":"ns-beta"}]}`
	wrapped := `{"data":{"data":"` + strings.ReplaceAll(strings.ReplaceAll(innerJSON, `\`, `\\`), `"`, `\"`) + `"},"type":"111"}`

	tests := []testCase{
		{
			name: "process_text_with_truncated_tail",
			input: "先检查CLI工具是否可用。\n" +
				"确认可用后查看record-list命令。\n" +
				"```json\n" +
				`{"data":[{"name":"svc-alpha","spaceName":"ns-alpha"},{"name":"svc-beta-consumer","spaceName":"ns-beta"},{"name":"svc-gamma","spaceName":"ns-beta","manager":"` +
				"\n```",
			expectSuccess: true,
			rowAssert: &rowAssertion{
				minRows: 2,
				names:   []string{"svc-alpha", "svc-beta-consumer"},
			},
		},
		{
			name:             "wrapped_data_field_json_string",
			input:            wrapped,
			expectSuccess:    true,
			expectStrategyIn: "embedded",
			rowAssert: &rowAssertion{
				minRows: 2,
				names:   []string{"svc-alpha", "svc-delta-consumer"},
			},
		},
		{
			name:          "repair_only_json_still_recovers",
			input:         `const result = {'name':'demo','n':1,};`,
			expectSuccess: true,
		},
		{
			name:          "non_json_text_should_fail",
			input:         "这是纯说明文本，没有任何结构化数据。",
			expectSuccess: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := parseJsonWithFixes(tc.input, "")
			if tc.expectSuccess != got.Success {
				t.Fatalf("unexpected success=%v, error=%s result=%#v", got.Success, got.Error, got.ExtractedJson)
			}
			if !tc.expectSuccess {
				return
			}
			if tc.expectStrategyIn != "" && !strings.Contains(got.SourceStrategy, tc.expectStrategyIn) {
				t.Fatalf("expected source strategy containing %q, got: %s", tc.expectStrategyIn, got.SourceStrategy)
			}
			if tc.rowAssert == nil {
				return
			}
			obj, ok := got.ExtractedJson.(map[string]interface{})
			if !ok {
				t.Fatalf("expected object json for row assertion, got: %#v", got.ExtractedJson)
			}
			rows, ok := obj["data"].([]interface{})
			if !ok || len(rows) < tc.rowAssert.minRows {
				t.Fatalf("expected rows >= %d, got: %#v", tc.rowAssert.minRows, obj["data"])
			}
			names := make(map[string]bool, len(rows))
			for _, row := range rows {
				m, ok := row.(map[string]interface{})
				if !ok {
					continue
				}
				if n, ok := m["name"].(string); ok {
					names[n] = true
				}
			}
			for _, name := range tc.rowAssert.names {
				if !names[name] {
					t.Fatalf("expected name %q in parsed data, got names=%#v", name, names)
				}
			}
		})
	}
}
