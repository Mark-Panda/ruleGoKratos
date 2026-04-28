package data

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/base"
	"github.com/rulego/rulego/utils/el"
	"github.com/rulego/rulego/utils/maps"
)

func init() {
	_ = rulego.Registry.Register(&JsonExtractComponent{})
}

type JsonExtractComponent struct {
	Config JsonExtractConfiguration

	sourceTpl         el.Template
	extractPatternTpl el.Template
	schemaPathsTpl    el.Template
	hasVar            bool
}

type JsonExtractConfiguration struct {
	Source         string `json:"source"`
	ExtractPattern string `json:"extractPattern"`
	ParseMode      string `json:"parseMode"`  // strict | auto | aggressive
	EmitReport     bool   `json:"emitReport"` // true 时输出提取与修复报告
	SchemaPaths    string `json:"schemaPaths"` // 逗号/分号/换行分隔，如 data[].name,data[].spaceName
}

func (c *JsonExtractComponent) New() types.Node {
	return &JsonExtractComponent{}
}

func (c *JsonExtractComponent) Type() string {
	return "x/jsonExtract"
}

func (c *JsonExtractComponent) Def() types.ComponentForm {
	return types.ComponentForm{
		Label: "JSON提取与纠错",
		Desc:  "从文本中提取 JSON 并做格式纠错与补全",
	}
}

func (c *JsonExtractComponent) Init(_ types.Config, configuration types.Configuration) error {
	if err := maps.Map2Struct(configuration, &c.Config); err != nil {
		return err
	}

	tpls := map[*el.Template]string{
		&c.sourceTpl:         c.Config.Source,
		&c.extractPatternTpl: c.Config.ExtractPattern,
		&c.schemaPathsTpl:    c.Config.SchemaPaths,
	}

	for tpl, s := range tpls {
		t, err := el.NewTemplate(s)
		if err != nil {
			return err
		}
		*tpl = t
		if t.HasVar() {
			c.hasVar = true
		}
	}

	return nil
}

func (c *JsonExtractComponent) OnMsg(ctx types.RuleContext, msg types.RuleMsg) {
	evn := base.NodeUtils.GetEvnAndMetadata(ctx, msg)
	if evn == nil {
		evn = map[string]interface{}{}
	}

	// 优先使用 source 配置字段（支持模板变量），为空时回退到 msg.Data
	inputText := strings.TrimSpace(c.sourceTpl.ExecuteAsString(evn))
	if inputText == "" {
		inputText = strings.TrimSpace(msg.GetData())
	}

	if inputText == "" {
		ctx.TellFailure(msg, errors.New("输入文本不能为空"))
		return
	}

	extractMode := strings.TrimSpace(c.extractPatternTpl.ExecuteAsString(evn))
	if extractMode == "" {
		extractMode = strings.TrimSpace(c.Config.ExtractPattern)
	}
	repairMode := strings.TrimSpace(c.Config.ParseMode)
	if repairMode == "" {
		repairMode = "auto"
	}
	schemaRaw := strings.TrimSpace(c.schemaPathsTpl.ExecuteAsString(evn))
	opts := ParseOptions{
		SchemaPaths: parseSchemaPathList(schemaRaw),
		ExtractMode: extractMode,
		RepairMode:  repairMode,
	}

	result := parseJsonWithFixesWithOptions(inputText, "", opts)
	if result.Success {
		payload := result.ExtractedJson
		if c.Config.EmitReport {
			payload = buildRepairReport(result)
		}
		if payload != nil {
			jsonBytes, _ := json.Marshal(payload)
			msg.SetData(string(jsonBytes))
		} else {
			msg.SetData(result.Result)
		}
		ctx.TellSuccess(msg)
	} else {
		msg.SetData(result.Error)
		ctx.TellFailure(msg, errors.New(result.Error))
	}
}

func (c *JsonExtractComponent) Destroy() {}
func (c *JsonExtractComponent) Close() error {
	return nil
}

type Result struct {
	Success          bool
	Result           string
	ExtractedJson    interface{}
	Error            string
	SourceStrategy   string
	RepairStrategies []string
	CandidateScore   int
	SchemaMatched    int
	SchemaMissing    []string
}

func parseJsonWithFixes(text, mode string) Result {
	return parseJsonWithFixesWithOptions(text, mode, ParseOptions{})
}

type ParseOptions struct {
	SchemaPaths  []string
	ExtractMode  string // auto | json | md
	RepairMode   string // strict | auto | aggressive
}

func parseJsonWithFixesWithOptions(text, mode string, opts ParseOptions) Result {
	trimText := strings.TrimSpace(text)
	if trimText == "" {
		return Result{Success: false, Error: "输入文本不能为空"}
	}
	trimText = normalizeCommonText(trimText)
	opts.ExtractMode, opts.RepairMode = normalizeModes(mode, opts.ExtractMode, opts.RepairMode)
	allowAggressive := opts.RepairMode == "aggressive"
	compiledSchemas := compileSchemaPaths(opts.SchemaPaths)

	// 1. 组织候选 JSON 文本（优先直接文本，其次 markdown 代码块与混杂文本片段提取）。
	type candidate struct {
		value    string
		strategy string
	}
	candidates := make([]candidate, 0, 12)
	appendCandidate := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		for _, existed := range candidates {
			if existed.value == v {
				return
			}
		}
		candidates = append(candidates, candidate{value: v, strategy: "direct"})
	}
	appendCandidateWithStrategy := func(v string, strategy string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		for _, existed := range candidates {
			if existed.value == v {
				return
			}
		}
		candidates = append(candidates, candidate{value: v, strategy: strategy})
	}

	appendCandidate(trimText)
	if opts.ExtractMode == "md" || opts.ExtractMode == "auto" {
		appendCandidateWithStrategy(extractJsonFromMarkdown(trimText), "markdown_fence")
	}
	if opts.ExtractMode == "json" || opts.ExtractMode == "auto" {
		fragments := extractBalancedJSONFragments(trimText, 4)
		for _, f := range fragments {
			appendCandidateWithStrategy(f, "balanced_fragment")
		}
		appendCandidateWithStrategy(extractJsonFromTaggedBlock(trimText), "tagged_block")
		appendCandidateWithStrategy(extractJsonFromAssignment(trimText), "assignment_rhs")
	}

	// 2. 对每个候选应用解析与修复策略。
	tryParse := func(v string) (interface{}, bool) {
		var parsed interface{}
		if err := json.Unmarshal([]byte(v), &parsed); err == nil {
			return parsed, true
		}
		return nil, false
	}
	successes := make([]Result, 0, 8)
	for _, c := range candidates {
		var variants []string
		var variantStrategy []string
		if opts.RepairMode == "strict" {
			variants = []string{
				c.value,
				unwrapJSONStringCandidate(c.value),
			}
			variantStrategy = []string{
				"none",
				"unwrap_json_string",
			}
		} else {
			variants = []string{
				c.value,
				unwrapJSONStringCandidate(c.value),
				fixJsonFormat(c.value),
				completeJson(c.value),
				completeJson(fixJsonFormat(c.value)),
				unwrapJSONStringCandidate(fixJsonFormat(c.value)),
				fixJsonFormat(completeJson(c.value)),
			}
			variantStrategy = []string{
				"none",
				"unwrap_json_string",
				"fix_json_format",
				"complete_json",
				"fix_then_complete",
				"fix_then_unwrap",
				"complete_then_fix",
			}
		}
		if allowAggressive {
			aggressive := make([]string, 0, len(variants))
			for _, v := range variants {
				aggressive = append(aggressive, aggressiveNormalize(v))
			}
			for i, v := range aggressive {
				if strings.TrimSpace(v) != "" {
					variants = append(variants, v)
					variantStrategy = append(variantStrategy, "aggressive_normalize("+variantStrategy[i]+")")
				}
			}
		}
		var primitiveFallback interface{}
		hasPrimitiveFallback := false
		for idx, item := range variants {
			item = strings.TrimSpace(item)
			if item == "" || !isValidJSON(item) {
				continue
			}
			if parsed, ok := tryParse(item); ok {
				switch parsed.(type) {
				case map[string]interface{}, []interface{}:
					repairs := make([]string, 0, 2)
					if idx < len(variantStrategy) && variantStrategy[idx] != "none" {
						repairs = append(repairs, variantStrategy[idx])
					}
					enriched, schemaMatched, schemaMissing, schemaChanged := applySchemaCompletion(parsed, compiledSchemas)
					if schemaChanged {
						repairs = append(repairs, "schema_complete")
					}
					score := scoreParsedCandidate(enriched, c.strategy, repairs, schemaMatched, len(schemaMissing))
					successes = append(successes, Result{
						Success:          true,
						Result:           formatJSON(enriched),
						ExtractedJson:    enriched,
						SourceStrategy:   c.strategy,
						RepairStrategies: repairs,
						CandidateScore:   score,
						SchemaMatched:    schemaMatched,
						SchemaMissing:    schemaMissing,
					})
				default:
					if !hasPrimitiveFallback {
						primitiveFallback = parsed
						hasPrimitiveFallback = true
					}
				}
			}
		}
		if hasPrimitiveFallback {
			successes = append(successes, Result{
				Success:          true,
				Result:           formatJSON(primitiveFallback),
				ExtractedJson:    primitiveFallback,
				SourceStrategy:   c.strategy,
				RepairStrategies: []string{"primitive_fallback"},
				CandidateScore:   scoreParsedCandidate(primitiveFallback, c.strategy, []string{"primitive_fallback"}, 0, len(compiledSchemas)),
				SchemaMissing:    append([]string(nil), opts.SchemaPaths...),
			})
		}
	}
	if len(successes) > 0 {
		return selectBestResult(successes)
	}

	return Result{Success: false, Error: "无法解析 JSON，请检查输入格式是否正确"}
}

func normalizeParseMode(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "", "auto", "md":
		return "auto"
	case "strict":
		return "strict"
	case "aggressive":
		return "aggressive"
	default:
		return "auto"
	}
}

func normalizeExtractMode(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case "", "auto":
		return "auto"
	case "json":
		return "json"
	case "md":
		return "md"
	default:
		return "auto"
	}
}

// normalizeModes 兼容历史：parseJsonWithFixes(text, mode) 传单参数时，mode 可能是修复模式或提取模式。
func normalizeModes(legacyMode string, extractMode string, repairMode string) (string, string) {
	em := normalizeExtractMode(extractMode)
	rm := normalizeParseMode(repairMode)

	legacy := strings.ToLower(strings.TrimSpace(legacyMode))
	if legacy == "" {
		return em, rm
	}
	// 旧调用：mode 既可能是 strict/auto/aggressive，也可能是 md/json/auto
	switch legacy {
	case "strict", "aggressive":
		return em, legacy
	case "md", "json":
		return legacy, rm
	case "auto":
		// auto 对两者都可作为默认值；仅在未显式配置时覆盖
		if strings.TrimSpace(extractMode) == "" {
			em = "auto"
		}
		if strings.TrimSpace(repairMode) == "" {
			rm = "auto"
		}
		return em, rm
	default:
		return em, rm
	}
}

type schemaSegment struct {
	Key     string
	IsArray bool
}

type compiledSchemaPath struct {
	Raw      string
	Segments []schemaSegment
}

func parseSchemaPathList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := regexp.MustCompile(`[\n,;]+`).Split(raw, -1)
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

func compileSchemaPaths(paths []string) []compiledSchemaPath {
	if len(paths) == 0 {
		return nil
	}
	out := make([]compiledSchemaPath, 0, len(paths))
	for _, raw := range paths {
		segmentsRaw := strings.Split(strings.TrimSpace(raw), ".")
		segments := make([]schemaSegment, 0, len(segmentsRaw))
		valid := true
		for _, segRaw := range segmentsRaw {
			segRaw = strings.TrimSpace(segRaw)
			if segRaw == "" {
				valid = false
				break
			}
			isArray := strings.HasSuffix(segRaw, "[]")
			key := strings.TrimSuffix(segRaw, "[]")
			key = strings.TrimSpace(key)
			if key == "" {
				valid = false
				break
			}
			segments = append(segments, schemaSegment{Key: key, IsArray: isArray})
		}
		if !valid || len(segments) == 0 {
			continue
		}
		out = append(out, compiledSchemaPath{Raw: raw, Segments: segments})
	}
	return out
}

func applySchemaCompletion(parsed interface{}, schemas []compiledSchemaPath) (interface{}, int, []string, bool) {
	if len(schemas) == 0 {
		return parsed, 0, nil, false
	}
	m, ok := parsed.(map[string]interface{})
	if !ok {
		return parsed, 0, schemaRawList(schemas), false
	}
	matched := 0
	missing := make([]string, 0, len(schemas))
	changed := false
	for _, schema := range schemas {
		exists := hasSchemaPath(m, schema.Segments)
		if exists {
			matched++
		} else {
			missing = append(missing, schema.Raw)
		}
		if ensureSchemaPath(m, schema.Segments) {
			changed = true
		}
	}
	return m, matched, missing, changed
}

func hasSchemaPath(node interface{}, segments []schemaSegment) bool {
	if len(segments) == 0 {
		return true
	}
	obj, ok := node.(map[string]interface{})
	if !ok {
		return false
	}
	seg := segments[0]
	value, exists := obj[seg.Key]
	if !exists {
		return false
	}
	if seg.IsArray {
		arr, ok := value.([]interface{})
		if !ok || len(arr) == 0 {
			return false
		}
		if len(segments) == 1 {
			return true
		}
		for _, item := range arr {
			if hasSchemaPath(item, segments[1:]) {
				return true
			}
		}
		return false
	}
	if len(segments) == 1 {
		return value != nil
	}
	return hasSchemaPath(value, segments[1:])
}

func ensureSchemaPath(node map[string]interface{}, segments []schemaSegment) bool {
	if len(segments) == 0 {
		return false
	}
	seg := segments[0]
	value, exists := node[seg.Key]
	changed := false
	if !exists {
		if seg.IsArray {
			node[seg.Key] = []interface{}{}
		} else if len(segments) == 1 {
			node[seg.Key] = nil
		} else {
			node[seg.Key] = map[string]interface{}{}
		}
		value = node[seg.Key]
		changed = true
	}
	if seg.IsArray {
		arr, ok := value.([]interface{})
		if !ok {
			arr = []interface{}{}
			node[seg.Key] = arr
			changed = true
		}
		if len(segments) == 1 {
			return changed
		}
		for i := range arr {
			itemMap, ok := arr[i].(map[string]interface{})
			if !ok {
				continue
			}
			if ensureSchemaPath(itemMap, segments[1:]) {
				changed = true
				arr[i] = itemMap
			}
		}
		node[seg.Key] = arr
		return changed
	}
	if len(segments) == 1 {
		return changed
	}
	childMap, ok := value.(map[string]interface{})
	if !ok {
		childMap = map[string]interface{}{}
		node[seg.Key] = childMap
		changed = true
	}
	if ensureSchemaPath(childMap, segments[1:]) {
		changed = true
		node[seg.Key] = childMap
	}
	return changed
}

func schemaRawList(paths []compiledSchemaPath) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		out = append(out, p.Raw)
	}
	return out
}

func scoreParsedCandidate(parsed interface{}, source string, repairs []string, schemaMatched int, schemaMissing int) int {
	score := 0
	switch parsed.(type) {
	case map[string]interface{}:
		score += 120
	case []interface{}:
		score += 110
	default:
		score += 25
	}
	score += minInt(countJSONNodes(parsed), 80)
	score += schemaMatched * 25
	score -= schemaMissing * 20
	switch source {
	case "direct":
		score += 8
	case "markdown_fence":
		score += 5
	case "balanced_fragment":
		score += 4
	case "tagged_block":
		score += 5
	case "assignment_rhs":
		score += 4
	}
	score -= len(repairs) * 2
	return score
}

func countJSONNodes(v interface{}) int {
	switch t := v.(type) {
	case map[string]interface{}:
		total := 1
		for _, child := range t {
			total += countJSONNodes(child)
		}
		return total
	case []interface{}:
		total := 1
		for _, child := range t {
			total += countJSONNodes(child)
		}
		return total
	default:
		return 1
	}
}

func selectBestResult(results []Result) Result {
	best := results[0]
	for i := 1; i < len(results); i++ {
		r := results[i]
		if r.CandidateScore > best.CandidateScore {
			best = r
			continue
		}
		if r.CandidateScore == best.CandidateScore {
			if len(r.SchemaMissing) < len(best.SchemaMissing) {
				best = r
				continue
			}
			if len(r.RepairStrategies) < len(best.RepairStrategies) {
				best = r
			}
		}
	}
	return best
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func normalizeCommonText(text string) string {
	text = strings.TrimSpace(strings.TrimPrefix(text, "\uFEFF"))
	replacer := strings.NewReplacer(
		"“", `"`,
		"”", `"`,
		"‘", "'",
		"’", "'",
		"，", ",",
		"：", ":",
		"；", ";",
		"（", "(",
		"）", ")",
		"｛", "{",
		"｝", "}",
		"［", "[",
		"］", "]",
	)
	return replacer.Replace(text)
}

func extractJsonFromMarkdown(text string) string {
	re := regexp.MustCompile("(?s)```(?:json)?\\s*([\\s\\S]*?)```")
	matches := re.FindStringSubmatch(text)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	re2 := regexp.MustCompile("(?s)```\\s*([\\s\\S]*?)```")
	matches2 := re2.FindStringSubmatch(text)
	if len(matches2) >= 2 {
		return strings.TrimSpace(matches2[1])
	}
	return ""
}

func extractJsonFromTaggedBlock(text string) string {
	re := regexp.MustCompile(`(?is)<(?:json|output|result)[^>]*>\s*([\s\S]*?)\s*</(?:json|output|result)>`)
	m := re.FindStringSubmatch(text)
	if len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func extractJsonFromAssignment(text string) string {
	re := regexp.MustCompile(`(?is)\b(?:const|let|var|return)?\s*(?:json|result|data)?\s*=?\s*([\{\[][^\n\r]*)`)
	m := re.FindStringSubmatch(text)
	if len(m) < 2 {
		return ""
	}
	candidate := strings.TrimSpace(m[1])
	frag, ok := cutBalancedJSONFragment(candidate)
	if ok {
		return strings.TrimSpace(frag)
	}
	return candidate
}

// extractBalancedJSONFragments 从混杂文本中提取平衡的 JSON 片段（对象或数组）。
// 仅做结构扫描，不要求片段本身已是合法 JSON，后续可结合修复策略解析。
func extractBalancedJSONFragments(text string, maxFragments int) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	if maxFragments <= 0 {
		maxFragments = 1
	}
	out := make([]string, 0, maxFragments)
	seen := make(map[string]struct{}, maxFragments)
	for i := 0; i < len(text); i++ {
		ch := text[i]
		if ch != '{' && ch != '[' {
			continue
		}
		if frag, ok := cutBalancedJSONFragment(text[i:]); ok {
			frag = strings.TrimSpace(frag)
			if frag == "" {
				continue
			}
			if _, exists := seen[frag]; exists {
				continue
			}
			seen[frag] = struct{}{}
			out = append(out, frag)
			if len(out) >= maxFragments {
				break
			}
		}
	}
	return out
}

func cutBalancedJSONFragment(text string) (string, bool) {
	if text == "" {
		return "", false
	}
	first := text[0]
	if first != '{' && first != '[' {
		return "", false
	}
	stack := make([]byte, 0, 8)
	if first == '{' {
		stack = append(stack, '}')
	} else {
		stack = append(stack, ']')
	}
	inString := false
	escaped := false

	for i := 1; i < len(text); i++ {
		ch := text[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{':
			stack = append(stack, '}')
		case '[':
			stack = append(stack, ']')
		case '}', ']':
			if len(stack) == 0 || ch != stack[len(stack)-1] {
				return "", false
			}
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				return text[:i+1], true
			}
		}
	}
	return "", false
}

func unwrapJSONStringCandidate(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return s
	}
	quote := s[0]
	if (quote != '"' && quote != '\'') || s[len(s)-1] != quote {
		return s
	}
	var unquoted string
	if quote == '"' {
		if err := json.Unmarshal([]byte(s), &unquoted); err != nil {
			return s
		}
	} else {
		unquoted = strings.ReplaceAll(s[1:len(s)-1], `\'`, `'`)
	}
	unquoted = strings.TrimSpace(unquoted)
	if strings.HasPrefix(unquoted, "{") || strings.HasPrefix(unquoted, "[") {
		return unquoted
	}
	return s
}

func fixJsonFormat(jsonStr string) string {
	fixed := normalizeCommonText(jsonStr)
	fixed = regexp.MustCompile(`(?m)//.*$`).ReplaceAllString(fixed, "")
	fixed = regexp.MustCompile(`(?s)/\*.*?\*/`).ReplaceAllString(fixed, "")
	fixed = strings.ReplaceAll(fixed, "'", "\"")
	fixed = regexp.MustCompile(`,(\s*[}\]])`).ReplaceAllString(fixed, "$1")
	fixed = regexp.MustCompile(`([{,\s]+)([a-zA-Z_][a-zA-Z0-9_]*)\s*:`).ReplaceAllString(fixed, `$1"$2":`)
	fixed = strings.ReplaceAll(fixed, "undefined", "null")
	fixed = regexp.MustCompile(`\bNone\b`).ReplaceAllString(fixed, "null")
	fixed = regexp.MustCompile(`\bTrue\b`).ReplaceAllString(fixed, "true")
	fixed = regexp.MustCompile(`\bFalse\b`).ReplaceAllString(fixed, "false")
	return fixed
}

func aggressiveNormalize(jsonStr string) string {
	fixed := strings.TrimSpace(jsonStr)
	if fixed == "" {
		return fixed
	}
	fixed = strings.TrimPrefix(fixed, "json")
	fixed = strings.TrimPrefix(fixed, "JSON")
	fixed = strings.TrimSpace(fixed)
	fixed = strings.TrimSuffix(fixed, ";")
	fixed = regexp.MustCompile(`\bNaN\b`).ReplaceAllString(fixed, "null")
	fixed = regexp.MustCompile(`\b-?Infinity\b`).ReplaceAllString(fixed, "null")
	return fixed
}

func completeJson(jsonStr string) string {
	s := strings.TrimSpace(jsonStr)
	if s == "" {
		return s
	}
	if !strings.HasPrefix(s, "{") && !strings.HasPrefix(s, "[") {
		return s
	}

	buf := make([]byte, 0, len(s)+8)
	stack := make([]byte, 0, 8)
	inString := false
	escaped := false

	push := func(ch byte) {
		stack = append(stack, ch)
	}
	pop := func(ch byte) bool {
		if len(stack) == 0 || stack[len(stack)-1] != ch {
			return false
		}
		stack = stack[:len(stack)-1]
		return true
	}

	for i := 0; i < len(s); i++ {
		ch := s[i]
		buf = append(buf, ch)
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{':
			push('}')
		case '[':
			push(']')
		case '}':
			if !pop('}') {
				buf = buf[:len(buf)-1]
			}
		case ']':
			if !pop(']') {
				buf = buf[:len(buf)-1]
			}
		}
	}

	out := strings.TrimSpace(string(buf))
	if inString {
		out += `"`
	}
	out = regexp.MustCompile(`,(\s*$)`).ReplaceAllString(out, "$1")
	out = regexp.MustCompile(`,(\s*[}\]])`).ReplaceAllString(out, "$1")
	for len(stack) > 0 {
		out = regexp.MustCompile(`,(\s*$)`).ReplaceAllString(out, "$1")
		out += string(stack[len(stack)-1])
		stack = stack[:len(stack)-1]
	}
	return out
}

func isValidJSON(s string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(s), &js) == nil
}

func formatJSON(v interface{}) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}

func buildRepairReport(result Result) interface{} {
	return map[string]interface{}{
		"success":           result.Success,
		"source_strategy":   result.SourceStrategy,
		"repair_strategies": result.RepairStrategies,
		"score":             result.CandidateScore,
		"schema_matched":    result.SchemaMatched,
		"schema_missing":    result.SchemaMissing,
		"data":              result.ExtractedJson,
	}
}
