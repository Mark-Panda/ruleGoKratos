package data

import (
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strconv"
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

	sourceTpl el.Template
	hasVar    bool
}

type JsonExtractConfiguration struct {
	Source string `json:"source"`
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
		Desc:  "从文本中提取 JSON（固定最严格模式）",
	}
}

func (c *JsonExtractComponent) Init(_ types.Config, configuration types.Configuration) error {
	if err := maps.Map2Struct(configuration, &c.Config); err != nil {
		return err
	}

	t, err := el.NewTemplate(c.Config.Source)
	if err != nil {
		return err
	}
	c.sourceTpl = t
	if t.HasVar() {
		c.hasVar = true
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

	result := parseJsonWithFixesWithOptions(inputText, "", ParseOptions{})
	if result.Success {
		normalized := normalizeTopLevelArrayToStringKeyMap(result.ExtractedJson)
		result.ExtractedJson = normalized
		jsonBytes, err := json.Marshal(normalized)
		if err != nil {
			msg.DataType = types.TEXT
			msg.SetData("JSON提取结果序列化失败")
			ctx.TellFailure(msg, err)
			return
		}
		msg.DataType = types.JSON
		msg.SetData(string(jsonBytes))
		ctx.TellSuccess(msg)
	} else {
		msg.DataType = types.TEXT
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
	TruncatedTailDropped bool
}

func parseJsonWithFixes(text, mode string) Result {
	return parseJsonWithFixesWithOptions(text, mode, ParseOptions{})
}

type ParseOptions struct{}

func parseJsonWithFixesWithOptions(text, mode string, opts ParseOptions) Result {
	_ = mode
	_ = opts
	trimText := strings.TrimSpace(text)
	if trimText == "" {
		return Result{Success: false, Error: "输入文本不能为空"}
	}
	trimText = normalizeCommonText(trimText)

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
	appendCandidateWithStrategy(extractFromFirstJSONStart(trimText), "first_json_start")
	embeddedTexts := extractEmbeddedTextCandidatesFromJSON(trimText, 8)
	for _, embedded := range embeddedTexts {
		appendCandidateWithStrategy(embedded, "embedded_text_field")
		appendCandidateWithStrategy(extractFromFirstJSONStart(embedded), "embedded_first_json_start")
		appendCandidateWithStrategy(extractJsonFromMarkdown(embedded), "embedded_markdown_fence")
		fragments := extractBalancedJSONFragments(embedded, 12)
		for _, f := range fragments {
			appendCandidateWithStrategy(f, "embedded_balanced_fragment")
		}
		appendCandidateWithStrategy(extractJsonFromTaggedBlock(embedded), "embedded_tagged_block")
		appendCandidateWithStrategy(extractJsonFromAssignment(embedded), "embedded_assignment_rhs")
	}
	appendCandidateWithStrategy(extractJsonFromMarkdown(trimText), "markdown_fence")
	fragments := extractBalancedJSONFragments(trimText, 20)
	for _, f := range fragments {
		appendCandidateWithStrategy(f, "balanced_fragment")
	}
	appendCandidateWithStrategy(extractJsonFromTaggedBlock(trimText), "tagged_block")
	appendCandidateWithStrategy(extractJsonFromAssignment(trimText), "assignment_rhs")

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
		completeOnly, completeOnlyDropped := completeJsonWithMeta(c.value)
		completeAfterFix, completeAfterFixDropped := completeJsonWithMeta(fixJsonFormat(c.value))
		variants := []string{
			c.value,
			unwrapJSONStringCandidate(c.value),
			fixJsonFormat(c.value),
			completeOnly,
			completeAfterFix,
			unwrapJSONStringCandidate(fixJsonFormat(c.value)),
			fixJsonFormat(completeOnly),
		}
		variantStrategy := []string{
			"none",
			"unwrap_json_string",
			"fix_json_format",
			"complete_json",
			"fix_then_complete",
			"fix_then_unwrap",
			"complete_then_fix",
		}
		variantTailDropped := []bool{
			false,
			false,
			false,
			completeOnlyDropped,
			completeAfterFixDropped,
			false,
			completeOnlyDropped,
		}
		aggressive := make([]string, 0, len(variants))
		for _, v := range variants {
			aggressive = append(aggressive, aggressiveNormalize(v))
		}
		for i, v := range aggressive {
			if strings.TrimSpace(v) == "" {
				continue
			}
			variants = append(variants, v)
			variantStrategy = append(variantStrategy, "aggressive_normalize("+variantStrategy[i]+")")
			if i < len(variantTailDropped) {
				variantTailDropped = append(variantTailDropped, variantTailDropped[i])
			} else {
				variantTailDropped = append(variantTailDropped, false)
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
					tailDropped := idx < len(variantTailDropped) && variantTailDropped[idx]
					if tailDropped {
						repairs = append(repairs, "truncated_tail_dropped")
					}
					score := scoreParsedCandidate(parsed, c.strategy, repairs)
					successes = append(successes, Result{
						Success:          true,
						Result:           formatJSON(parsed),
						ExtractedJson:    parsed,
						SourceStrategy:   c.strategy,
						RepairStrategies: repairs,
						CandidateScore:   score,
						TruncatedTailDropped: tailDropped,
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
				CandidateScore:   scoreParsedCandidate(primitiveFallback, c.strategy, []string{"primitive_fallback"}),
			})
		}
	}
	if len(successes) > 0 {
		return selectBestResult(successes)
	}

	return Result{Success: false, Error: "无法解析 JSON，请检查输入格式是否正确"}
}

type schemaSegment struct {
	Key     string
	IsArray bool
}

type compiledSchemaPath struct {
	Raw      string
	Segments []schemaSegment
}

//lint:ignore U1000 "kept for future use"
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

func scoreParsedCandidate(parsed interface{}, source string, repairs []string) int {
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
	if hasTopLevelDataArray(parsed) {
		score += 18
	}
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
	case "first_json_start", "embedded_first_json_start":
		score += 7
	}
	score -= len(repairs) * 2
	return score
}

func hasTopLevelDataArray(v interface{}) bool {
	m, ok := v.(map[string]interface{})
	if !ok {
		return false
	}
	data, ok := m["data"]
	if !ok {
		return false
	}
	_, ok = data.([]interface{})
	return ok
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

func extractFromFirstJSONStart(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	firstObj := strings.IndexByte(text, '{')
	firstArr := strings.IndexByte(text, '[')
	start := -1
	switch {
	case firstObj >= 0 && firstArr >= 0:
		if firstObj < firstArr {
			start = firstObj
		} else {
			start = firstArr
		}
	case firstObj >= 0:
		start = firstObj
	case firstArr >= 0:
		start = firstArr
	default:
		return ""
	}
	if start < 0 || start >= len(text) {
		return ""
	}
	return strings.TrimSpace(text[start:])
}

// extractEmbeddedTextCandidatesFromJSON 用于处理 source 指向对象时的场景（如 ${msg.data}），
// 自动从 JSON 容器中抽取可能包含目标结构化内容的文本字段。
func extractEmbeddedTextCandidatesFromJSON(text string, max int) []string {
	if strings.TrimSpace(text) == "" || max <= 0 {
		return nil
	}
	var parsed interface{}
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return nil
	}

	type textCandidate struct {
		value string
		score int
	}
	candidates := make([]textCandidate, 0, 8)
	seen := make(map[string]struct{}, 16)

	scoreText := func(s string) int {
		score := 0
		if strings.Contains(s, "```") {
			score += 120
		}
		if strings.Contains(s, `{"`) || strings.Contains(s, `["`) {
			score += 45
		}
		if strings.Contains(s, "{") || strings.Contains(s, "[") {
			score += 30
		}
		if strings.Contains(strings.ToLower(s), "json") {
			score += 15
		}
		score += minInt(len(s)/64, 80)
		return score
	}

	var walk func(v interface{}, depth int)
	walk = func(v interface{}, depth int) {
		if depth > 8 || len(candidates) >= max*3 {
			return
		}
		switch t := v.(type) {
		case map[string]interface{}:
			for _, child := range t {
				walk(child, depth+1)
			}
		case []interface{}:
			for _, child := range t {
				walk(child, depth+1)
			}
		case string:
			s := strings.TrimSpace(t)
			if len(s) < 8 {
				return
			}
			if _, ok := seen[s]; ok {
				return
			}
			seen[s] = struct{}{}
			candidates = append(candidates, textCandidate{
				value: s,
				score: scoreText(s),
			})
		}
	}
	walk(parsed, 0)
	if len(candidates) == 0 {
		return nil
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return len(candidates[i].value) > len(candidates[j].value)
		}
		return candidates[i].score > candidates[j].score
	})
	if len(candidates) > max {
		candidates = candidates[:max]
	}
	out := make([]string, 0, len(candidates))
	for _, c := range candidates {
		out = append(out, c.value)
	}
	return out
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
	all := make([]string, 0, maxFragments*2)
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
			all = append(all, frag)
		}
	}
	if len(all) <= maxFragments {
		return all
	}
	sort.SliceStable(all, func(i, j int) bool {
		if len(all[i]) == len(all[j]) {
			return i < j
		}
		return len(all[i]) > len(all[j])
	})
	return all[:maxFragments]
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

//lint:ignore U1000 "kept for future use"
func completeJson(jsonStr string) string {
	out, _ := completeJsonWithMeta(jsonStr)
	return out
}

func completeJsonWithMeta(jsonStr string) (string, bool) {
	s := strings.TrimSpace(jsonStr)
	if s == "" {
		return s, false
	}
	if !strings.HasPrefix(s, "{") && !strings.HasPrefix(s, "[") {
		return s, false
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
	return healCompletedJSONTail(out)
}

// healCompletedJSONTail 在 completeJson 的基础上，修复因截断导致的尾部悬空片段。
// 典型场景：...,"name":"study-statistics", "   -> 可能被补成 ...,""
// 该函数仅在原结果非合法 JSON 时启用启发式修复，避免影响正常结果。
func healCompletedJSONTail(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" || isValidJSON(s) {
		return s, false
	}
	candidates := []string{
		// 移除尾部空 key（由截断补全导致）
		regexp.MustCompile(`,\s*""\s*([}\]])`).ReplaceAllString(s, `$1`),
		// 移除尾部未完成 key（有 key 无冒号）
		regexp.MustCompile(`,\s*"[^"]*"\s*([}\]])`).ReplaceAllString(s, `$1`),
		// 移除尾部未完成 key:value（有冒号无值）
		regexp.MustCompile(`,\s*"[^"]*"\s*:\s*([}\]])`).ReplaceAllString(s, `$1`),
		// 移除闭合符前多余逗号
		regexp.MustCompile(`,\s*([}\]])`).ReplaceAllString(s, `$1`),
	}
	for _, c := range candidates {
		c = strings.TrimSpace(c)
		if c != "" && isValidJSON(c) {
			return c, c != s
		}
	}
	// 尝试串联修复一次，覆盖多重尾部噪声场景。
	chained := s
	chained = regexp.MustCompile(`,\s*""\s*([}\]])`).ReplaceAllString(chained, `$1`)
	chained = regexp.MustCompile(`,\s*"[^"]*"\s*([}\]])`).ReplaceAllString(chained, `$1`)
	chained = regexp.MustCompile(`,\s*"[^"]*"\s*:\s*([}\]])`).ReplaceAllString(chained, `$1`)
	chained = regexp.MustCompile(`,\s*([}\]])`).ReplaceAllString(chained, `$1`)
	chained = strings.TrimSpace(chained)
	if chained != "" && isValidJSON(chained) {
		return chained, chained != s
	}
	return s, false
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
		"truncated_tail_dropped": result.TruncatedTailDropped,
		"data":              result.ExtractedJson,
	}
}

// normalizeTopLevelArrayToStringKeyMap 将顶层 JSON 数组转为字符串 key 对象：
// [a,b] => {"0":a,"1":b}
// 仅转换顶层，避免破坏对象内字段的数组语义（如 data[]）。
func normalizeTopLevelArrayToStringKeyMap(v interface{}) interface{} {
	arr, ok := v.([]interface{})
	if !ok {
		return v
	}
	out := make(map[string]interface{}, len(arr))
	for i, item := range arr {
		out[strconv.Itoa(i)] = item
	}
	return out
}
