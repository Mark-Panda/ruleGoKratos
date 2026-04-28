package data

import "testing"

func TestParseJsonWithFixes_ExtractFromThinkWrappedText(t *testing.T) {
	input := `<think> 用户想法 </think>
我来帮你处理，先分析...
{"data":[{"name":"channel-platform-server","spaceName":"teacherschool","manager":"张一明","language":"Node","framework":"koa2"}]}`

	got := parseJsonWithFixes(input, "auto")
	if !got.Success {
		t.Fatalf("expected success, got error: %s", got.Error)
	}
	obj, ok := got.ExtractedJson.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object json, got: %#v", got.ExtractedJson)
	}
	data, ok := obj["data"].([]interface{})
	if !ok || len(data) != 1 {
		t.Fatalf("expected data array with one item, got: %#v", obj["data"])
	}
}

func TestParseJsonWithFixes_ExtractFromMarkdownFence(t *testing.T) {
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

func TestExtractFirstValidJSONFragment_Array(t *testing.T) {
	input := "prefix <think>xxx</think> [1,2,{\"a\":\"b\"}] suffix"
	fragments := extractBalancedJSONFragments(input, 2)
	if len(fragments) == 0 {
		t.Fatal("expected non-empty extracted fragments")
	}
	if fragments[0] != `[1,2,{"a":"b"}]` {
		t.Fatalf("unexpected fragment: %s", fragments[0])
	}
}

func TestParseJsonWithFixes_PythonLiteralsAndComments(t *testing.T) {
	input := `{
  // line comment
  "ok": True,
  "err": None,
  /* block comment */
  "items": [1,2,3,],
}`
	got := parseJsonWithFixes(input, "auto")
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
	if obj["err"] != nil {
		t.Fatalf("expected err=nil, got: %#v", obj["err"])
	}
}

func TestParseJsonWithFixes_CompleteTruncatedJSON(t *testing.T) {
	input := `{"data":[{"name":"a","lang":"Go"}, {"name":"b","lang":"Node"}`
	got := parseJsonWithFixes(input, "auto")
	if !got.Success {
		t.Fatalf("expected success, got error: %s", got.Error)
	}
	obj, ok := got.ExtractedJson.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object json, got: %#v", got.ExtractedJson)
	}
	items, ok := obj["data"].([]interface{})
	if !ok || len(items) != 2 {
		t.Fatalf("expected data with 2 items, got: %#v", obj["data"])
	}
}

func TestParseJsonWithFixes_DoubleEncodedJSONString(t *testing.T) {
	input := `"{\"data\":[{\"name\":\"svc-a\",\"framework\":\"kratos\"}]}"` // LLM 常见的 JSON 字符串包裹
	got := parseJsonWithFixes(input, "auto")
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

func TestParseJsonWithFixes_TaggedBlock(t *testing.T) {
	input := `<result>{"ok":true,"count":2}</result>`
	got := parseJsonWithFixes(input, "auto")
	if !got.Success {
		t.Fatalf("expected success, got error: %s", got.Error)
	}
	if got.SourceStrategy != "tagged_block" && got.SourceStrategy != "direct" && got.SourceStrategy != "balanced_fragment" {
		t.Fatalf("unexpected source strategy: %s", got.SourceStrategy)
	}
}

func TestParseJsonWithFixes_AssignmentAndSemicolon(t *testing.T) {
	input := `const result = {"name":"demo","n":1};`
	got := parseJsonWithFixes(input, "aggressive")
	if !got.Success {
		t.Fatalf("expected success, got error: %s", got.Error)
	}
	obj, ok := got.ExtractedJson.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object json, got: %#v", got.ExtractedJson)
	}
	if obj["name"] != "demo" {
		t.Fatalf("unexpected name: %#v", obj["name"])
	}
}

func TestBuildRepairReport(t *testing.T) {
	r := Result{
		Success:          true,
		SourceStrategy:   "balanced_fragment",
		RepairStrategies: []string{"fix_json_format"},
		ExtractedJson:    map[string]interface{}{"ok": true},
	}
	report, ok := buildRepairReport(r).(map[string]interface{})
	if !ok {
		t.Fatal("expected report map")
	}
	if report["source_strategy"] != "balanced_fragment" {
		t.Fatalf("unexpected source strategy: %#v", report["source_strategy"])
	}
}

func TestParseJsonWithFixesWithOptions_SchemaCompletion(t *testing.T) {
	input := `{"data":[{"name":"svc-a"}]}`
	got := parseJsonWithFixesWithOptions(input, "auto", ParseOptions{
		SchemaPaths: []string{
			"data[].name",
			"data[].spaceName",
			"data[].manager",
		},
	})
	if !got.Success {
		t.Fatalf("expected success, got error: %s", got.Error)
	}
	obj, ok := got.ExtractedJson.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object json, got: %#v", got.ExtractedJson)
	}
	data, ok := obj["data"].([]interface{})
	if !ok || len(data) != 1 {
		t.Fatalf("expected data with one item, got: %#v", obj["data"])
	}
	item, ok := data[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected map item, got: %#v", data[0])
	}
	if _, exists := item["spaceName"]; !exists {
		t.Fatalf("expected spaceName completed, got: %#v", item)
	}
	if _, exists := item["manager"]; !exists {
		t.Fatalf("expected manager completed, got: %#v", item)
	}
}

func TestParseJsonWithFixesWithOptions_SchemaScorePrefersMatchingCandidate(t *testing.T) {
	input := `前文 {"x":1} 后文 {"data":[{"name":"svc-a","spaceName":"ns"}]}`
	got := parseJsonWithFixesWithOptions(input, "auto", ParseOptions{
		SchemaPaths: []string{"data[].name", "data[].spaceName"},
	})
	if !got.Success {
		t.Fatalf("expected success, got error: %s", got.Error)
	}
	obj, ok := got.ExtractedJson.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object json, got: %#v", got.ExtractedJson)
	}
	if _, ok := obj["data"].([]interface{}); !ok {
		t.Fatalf("expected candidate with data selected, got: %#v", obj)
	}
}

func TestParseSchemaPathList(t *testing.T) {
	raw := "data[].name, data[].spaceName;\nmeta.total\n data[].name "
	got := parseSchemaPathList(raw)
	if len(got) != 3 {
		t.Fatalf("expected 3 unique paths, got %d: %#v", len(got), got)
	}
}

func TestParseJsonWithFixesWithOptions_StrictStillExtractsFragment(t *testing.T) {
	input := `前缀说明文本 {"data":[{"name":"svc-a","spaceName":"teacherschool"}]} 后缀`
	got := parseJsonWithFixesWithOptions(input, "", ParseOptions{
		ExtractMode: "json",
		RepairMode:  "strict",
	})
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

func TestNormalizeModesLegacyCompatibility(t *testing.T) {
	em, rm := normalizeModes("strict", "", "")
	if em != "auto" || rm != "strict" {
		t.Fatalf("unexpected legacy strict modes: extract=%s repair=%s", em, rm)
	}

	em, rm = normalizeModes("md", "", "")
	if em != "md" || rm != "auto" {
		t.Fatalf("unexpected legacy md modes: extract=%s repair=%s", em, rm)
	}
}
