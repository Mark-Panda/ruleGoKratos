package data

import (
	"strconv"
	"strings"
	"testing"
)

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
		t.Fatalf("expected data array with one item, got data=%#v obj=%#v", obj["data"], obj)
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

func TestParseJsonWithFixesWithOptions_MdModeFallbackWithoutFence(t *testing.T) {
	input := `<think>一些说明文字</think>
请输出如下结果：
{"data":[{"name":"teacher-ee","spaceName":"teacherschool","manager":"王刚","language":"Go","framework":"kratos"}]}`
	got := parseJsonWithFixesWithOptions(input, "", ParseOptions{
		ExtractMode: "md",
		RepairMode:  "auto",
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
		t.Fatalf("expected data array, got: %#v", obj["data"])
	}
}

func TestExtractBalancedJSONFragments_PrefersLargerFragments(t *testing.T) {
	input := strings.Join([]string{
		"prefix",
		`{"a":1}`,
		`{"b":2}`,
		`{"c":3}`,
		`{"d":4}`,
		`{"data":[{"name":"svc-a","spaceName":"ns-a"},{"name":"svc-b","spaceName":"ns-b"}]}`,
	}, " ")
	fragments := extractBalancedJSONFragments(input, 3)
	if len(fragments) == 0 {
		t.Fatal("expected non-empty fragments")
	}
	if !strings.Contains(fragments[0], `"data":[`) {
		t.Fatalf("expected largest data fragment selected first, got: %s", fragments[0])
	}
}

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

func TestParseJsonWithFixes_CompleteTruncatedTailDanglingKey(t *testing.T) {
	input := `{"data":[
{"name":"teacher-ee","spaceName":"teacherschool"},
{"name":"study-statistics", "`

	got := parseJsonWithFixesWithOptions(input, "auto", ParseOptions{
		ExtractMode: "json",
		RepairMode:  "auto",
		SchemaPaths: []string{"data[].name"},
	})
	if !got.Success {
		t.Fatalf("expected success, got error: %s", got.Error)
	}
	obj, ok := got.ExtractedJson.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object json, got: %#v", got.ExtractedJson)
	}
	data, ok := obj["data"].([]interface{})
	if !ok || len(data) != 2 {
		t.Fatalf("expected data with 2 items, got: %#v", obj["data"])
	}
	last, ok := data[1].(map[string]interface{})
	if !ok {
		t.Fatalf("expected second item object, got: %#v", data[1])
	}
	if last["name"] != "study-statistics" {
		t.Fatalf("expected second item name preserved, got: %#v", last["name"])
	}
	if !got.TruncatedTailDropped {
		t.Fatalf("expected truncated tail dropped marker, got: %#v", got.TruncatedTailDropped)
	}
	found := false
	for _, s := range got.RepairStrategies {
		if s == "truncated_tail_dropped" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected repair strategy includes truncated_tail_dropped, got: %#v", got.RepairStrategies)
	}
}

func TestBuildRepairReportIncludesTruncatedTailDropped(t *testing.T) {
	r := Result{
		Success:              true,
		SourceStrategy:       "balanced_fragment",
		RepairStrategies:     []string{"complete_json", "truncated_tail_dropped"},
		TruncatedTailDropped: true,
		ExtractedJson:        map[string]interface{}{"ok": true},
	}
	report, ok := buildRepairReport(r).(map[string]interface{})
	if !ok {
		t.Fatal("expected report map")
	}
	flag, ok := report["truncated_tail_dropped"].(bool)
	if !ok || !flag {
		t.Fatalf("expected truncated_tail_dropped=true, got: %#v", report["truncated_tail_dropped"])
	}
}

func TestParseJsonWithFixes_LargeFeishuBackendListTruncatedTail(t *testing.T) {
	input := "成功从飞书多维表格中提取了所有后端应用的信息，共 **128个** 后端服务：\n```json\n" + `{
  "data": [
    {"name":"teacher-ee","spaceName":"teacherschool","manager":"王刚","language":"Go","framework":"kratos"},
    {"name":"teacher-openapi-hunan","spaceName":"teacherschool","manager":"王刚","language":"Go","framework":"echo"},
    {"name":"pressuremock","spaceName":"teacherschool","manager":"","language":"","framework":""},
    {"name":"channel-core","spaceName":"teacherschool","manager":"张一明","language":"Go","framework":"kratos"},
    {"name":"volc-cloud-monitor-exporter","spaceName":"ops","manager":"张乾","language":"","framework":""},
    {"name":"achievement","spaceName":"7to12","manager":"李保川","language":"Go","framework":"kratos"},
    {"name":"achievement-admin","spaceName":"7to12","manager":"李保川","language":"Go","framework":"kratos"},
    {"name":"activity","spaceName":"7to12","manager":"徐斌","language":"Go","framework":"kratos"},
    {"name":"activity-consumer","spaceName":"7to12","manager":"闫鹏","language":"Go","framework":"kratos"},
    {"name":"ai-models-dkt","spaceName":"7to12","manager":"郭权威","language":"Go","framework":"kratos"},
    {"name":"backend-config","spaceName":"7to12","manager":"罗飞","language":"Go","framework":"kratos"},
    {"name":"comment","spaceName":"7to12","manager":"罗烽","language":"Go","framework":"gin"},
    {"name":"course-ai","spaceName":"7to12","manager":"郭权威","language":"Go","framework":"kratos"},
    {"name":"data-center-consumer","spaceName":"7to12","manager":"罗飞","language":"Go","framework":"gin"},
    {"name":"data-inspector-api","spaceName":"7to12","manager":"罗飞","language":"Go","framework":"kratos"},
    {"name":"data-inspector-slave","spaceName":"7to12","manager":"罗飞","language":"Go","framework":"kratos"},
    {"name":"desk-consumer","spaceName":"7to12","manager":"徐斌","language":"Go","framework":"gin"},
    {"name":"deskship","spaceName":"7to12","manager":"徐斌","language":"Go","framework":"gin"},
    {"name":"event-trigger-clear","spaceName":"7to12","manager":"徐斌","language":"Go","framework":"kratos"},
    {"name":"event-trigger-event","spaceName":"7to12","manager":"徐斌","language":"Go","framework":"kratos"},
    {"name":"event-trigger-trigger","spaceName":"7to12","manager":"徐斌","language":"Go","framework":"kratos"},
    {"name":"friend","spaceName":"7to12","manager":"刘阳（研发）","language":"Go","framework":"kratos"},
    {"name":"friend-consumer","spaceName":"7to12","manager":"刘阳（研发）","language":"Go","framework":"kratos"},
    {"name":"study-room","spaceName":"7to12","manager":"刘阳（研发）","language":"Go","framework":"kratos"},
    {"name":"study-room-cron","spaceName":"7to12","manager":"刘阳（研发）","language":"Go","framework":"kratos"},
    {"name":"study-search","spaceName":"7to12","manager":"李宝卫","language":"Go","framework":"gin"},
    {"name":"study-search-consumer","spaceName":"7to12","manager":"李宝卫","language":"Go","framework":"gin"},
    {"name":"study-statistics","spaceName":"7to12","manager":"` + "\n```"

	got := parseJsonWithFixesWithOptions(input, "auto", ParseOptions{
		ExtractMode: "md",
		RepairMode:  "auto",
		SchemaPaths: []string{"data[].name", "data[].spaceName"},
	})
	if !got.Success {
		t.Fatalf("expected success, got error: %s", got.Error)
	}
	obj, ok := got.ExtractedJson.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object json, got: %#v", got.ExtractedJson)
	}
	data, ok := obj["data"].([]interface{})
	if !ok {
		t.Fatalf("expected data array, got: %#v", obj["data"])
	}
	if len(data) < 20 {
		t.Fatalf("expected many parsed rows retained, got len=%d", len(data))
	}

	names := make(map[string]bool, len(data))
	for _, row := range data {
		m, ok := row.(map[string]interface{})
		if !ok {
			continue
		}
		if n, ok := m["name"].(string); ok {
			names[n] = true
		}
	}
	if !names["teacher-ee"] {
		t.Fatalf("expected teacher-ee present, names=%#v", names)
	}
	if !names["study-search-consumer"] {
		t.Fatalf("expected study-search-consumer present, names=%#v", names)
	}
	if !names["study-statistics"] {
		t.Fatalf("expected truncated row study-statistics repaired and kept, names=%#v", names)
	}
	// 末条记录的 manager 在输入中被截断，修复后会成为空字符串（字符串闭合补全）。
	var last map[string]interface{}
	for i := len(data) - 1; i >= 0; i-- {
		row, ok := data[i].(map[string]interface{})
		if !ok {
			continue
		}
		if row["name"] == "study-statistics" {
			last = row
			break
		}
	}
	if last == nil {
		t.Fatal("expected to find repaired study-statistics row")
	}
	if _, exists := last["manager"]; !exists {
		t.Fatalf("expected manager key exists after completion, row=%#v", last)
	}
	if last["manager"] != "" {
		t.Fatalf("expected manager empty string for repaired tail row, row=%#v", last)
	}
	t.Logf("parsed rows kept: %d", len(data))
}

func TestParseJsonWithFixes_WrappedMessageDataField(t *testing.T) {
	inner := `我需要先检查飞书CLI工具是否可用。
成功从飞书多维表格中提取后端服务：
` + "```json\n" + `{
  "data": [
    {"name":"teacher-ee","spaceName":"teacherschool","manager":"王刚","language":"Go","framework":"kratos"},
    {"name":"study-ai-agent-demo-consumer","spaceName":"7to12","manager":"刘阳（研发）","language":"Go","framework":"kratos"},
    {"name":"study-statistics","spaceName":"7to12","`
	outer := `{"data":{"data":` + strconv.Quote(inner) + `},"type":"111"}`

	got := parseJsonWithFixesWithOptions(outer, "auto", ParseOptions{
		ExtractMode: "auto",
		RepairMode:  "aggressive",
		SchemaPaths: []string{
			"data[].name",
			"data[].spaceName",
			"data[].manager",
			"data[].language",
			"data[].framework",
		},
	})
	if !got.Success {
		t.Fatalf("expected success, got error: %s", got.Error)
	}
	obj, ok := got.ExtractedJson.(map[string]interface{})
	if !ok {
		t.Fatalf("expected object json, got: %#v", got.ExtractedJson)
	}
	rows, ok := obj["data"].([]interface{})
	if !ok || len(rows) < 2 {
		t.Fatalf("expected parsed data array from embedded field, got: %#v", obj["data"])
	}
	first, ok := rows[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected first row map, got: %#v", rows[0])
	}
	if first["name"] != "teacher-ee" {
		t.Fatalf("expected first row teacher-ee, got: %#v", first["name"])
	}
}
