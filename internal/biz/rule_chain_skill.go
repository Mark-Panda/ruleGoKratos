package biz

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/rulego/rulego/api/types"
)

const defaultRuleChainSkillEntryFile = "SKILL.md"
const maxRuleChainSkillDirNameLength = 64
const defaultRuleChainSkillMsgType = "CHAIN"

var ruleChainSkillDirNameSanitizer = regexp.MustCompile(`[^a-z0-9]+`)

var errRuleChainSkillPathEscape = errors.New("skill 文件路径越界")

// RuleChainSkillStatus 表示规则链 Skill 的当前状态。
type RuleChainSkillStatus string

const (
	RuleChainSkillStatusMissing RuleChainSkillStatus = "missing"
	RuleChainSkillStatusStale   RuleChainSkillStatus = "stale"
	RuleChainSkillStatusReady   RuleChainSkillStatus = "ready"
)

// RuleChainSkillMeta 描述规则链 Skill 的持久化元数据。
type RuleChainSkillMeta struct {
	DirName                   string
	EntryFile                 string
	Signature                 string
	LastGenerated             string
	Status                    string
	GeneratedAt               string
	GeneratedByManagedAgentID int64
	LastError                 string
}

// ResolveRuleChainSkillStatus 根据文件存在性和签名判断 Skill 状态。
func ResolveRuleChainSkillStatus(skillRoot string, meta RuleChainSkillMeta, currentSignature string) (RuleChainSkillStatus, error) {
	dirName := normalizeRuleChainSkillDirName(meta.DirName)
	if dirName == "" {
		return RuleChainSkillStatusMissing, nil
	}

	entryFile := normalizeRuleChainSkillEntryFile(meta.EntryFile)

	skillEntryPath := filepath.Join(skillRoot, dirName, entryFile)
	if !isWithinSkillRoot(skillRoot, skillEntryPath) {
		return RuleChainSkillStatusMissing, nil
	}
	resolvedSkillEntryPath, err := resolveExistingRuleChainSkillPath(skillRoot, skillEntryPath)
	if err != nil {
		if os.IsNotExist(err) || errors.Is(err, errRuleChainSkillPathEscape) {
			return RuleChainSkillStatusMissing, nil
		}
		return "", err
	}
	content, err := os.ReadFile(resolvedSkillEntryPath)
	if err != nil {
		return "", err
	}
	if len(strings.TrimSpace(string(content))) == 0 {
		return RuleChainSkillStatusMissing, nil
	}

	generatedSignature := strings.TrimSpace(meta.Signature)
	if generatedSignature == "" {
		generatedSignature = strings.TrimSpace(meta.LastGenerated)
	}
	if generatedSignature != strings.TrimSpace(currentSignature) {
		return RuleChainSkillStatusStale, nil
	}
	if !strings.Contains(string(content), BuildRuleChainSkillSignatureAnchor(strings.TrimSpace(currentSignature))) {
		return RuleChainSkillStatusStale, nil
	}

	return RuleChainSkillStatusReady, nil
}

// RuleChainSkillPromptInput 为 Skill 生成提示词提供稳定输入。
type RuleChainSkillPromptInput struct {
	RuleChainID           string
	RuleChainName         string
	DirName               string
	SkillRoot             string
	MsgType               string
	Description           string
	RequestMetadataParams string
	RequestBodyParams     string
	ResponseBodyParams    string
}

// ParseRuleChainSkillMeta 从 configuration.flowgram.skill 读取持久化元数据。
func ParseRuleChainSkillMeta(configuration map[string]interface{}) RuleChainSkillMeta {
	flowgram := asJSONMap(configuration["flowgram"])
	skill := asJSONMap(flowgram["skill"])
	return RuleChainSkillMeta{
		DirName:                   stringFromJSONValue(skill["dir_name"]),
		EntryFile:                 stringFromJSONValue(skill["skill_entry_file"]),
		Signature:                 stringFromJSONValue(skill["signature"]),
		LastGenerated:             stringFromJSONValue(skill["signature"]),
		Status:                    stringFromJSONValue(skill["status"]),
		GeneratedAt:               stringFromJSONValue(skill["generated_at"]),
		GeneratedByManagedAgentID: int64FromJSONValue(skill["generated_by_managed_agent_id"]),
		LastError:                 stringFromJSONValue(skill["last_error"]),
	}
}

// BuildRuleChainSkillPromptInput 从规则链配置提取生成 Skill 所需内容。
func BuildRuleChainSkillPromptInput(ruleChain *types.RuleChain, dirName string) RuleChainSkillPromptInput {
	configuration := asJSONMap(ruleChain.RuleChain.Configuration)
	flowgram := asJSONMap(configuration["flowgram"])
	io := asJSONMap(flowgram["io"])
	description := stringFromJSONValue(ruleChain.RuleChain.AdditionalInfo["description"])
	if description == "" {
		description = stringFromJSONValue(flowgram["description"])
	}
	return RuleChainSkillPromptInput{
		RuleChainID:           strings.TrimSpace(ruleChain.RuleChain.ID),
		RuleChainName:         strings.TrimSpace(ruleChain.RuleChain.Name),
		DirName:               dirName,
		Description:           description,
		MsgType:               InferRuleChainSkillMsgType(ruleChain),
		RequestMetadataParams: marshalRuleChainSkillJSON(io["request_metadata_params"]),
		RequestBodyParams:     marshalRuleChainSkillJSON(io["request_message_body_params"]),
		ResponseBodyParams:    marshalRuleChainSkillJSON(io["response_message_body_params"]),
	}
}

// BuildRuleChainSkillMetaPatch 生成需要回写到 configuration.flowgram.skill 的最小字段集。
func BuildRuleChainSkillMetaPatch(meta RuleChainSkillMeta) map[string]interface{} {
	return map[string]interface{}{
		"flowgram": map[string]interface{}{
			"skill": map[string]interface{}{
				"dir_name":                      normalizeRuleChainSkillDirName(meta.DirName),
				"status":                        strings.TrimSpace(meta.Status),
				"signature":                     strings.TrimSpace(meta.Signature),
				"generated_at":                  strings.TrimSpace(meta.GeneratedAt),
				"generated_by_managed_agent_id": meta.GeneratedByManagedAgentID,
				"skill_entry_file":              normalizeRuleChainSkillEntryFile(meta.EntryFile),
				"last_error":                    strings.TrimSpace(meta.LastError),
			},
		},
	}
}

// ChooseRuleChainSkillDirName 优先复用已有 dir_name，否则基于名称/ID 生成稳定目录名。
func ChooseRuleChainSkillDirName(ruleChain *types.RuleChain, meta RuleChainSkillMeta) string {
	if dirName := normalizeRuleChainSkillDirName(meta.DirName); dirName != "" {
		return dirName
	}
	if dirName := BuildRuleChainSkillBaseDirName(ruleChain); dirName != "" {
		return dirName
	}
	return "rulechain-skill"
}

// ReadRuleChainSkillFile 确认生成文件存在且非空，并返回其内容。
func ReadRuleChainSkillFile(skillRoot string, dirName string, entryFile string) (string, error) {
	dirName = normalizeRuleChainSkillDirName(dirName)
	if dirName == "" {
		return "", fmt.Errorf("skill 目录名无效")
	}
	entryFile = normalizeRuleChainSkillEntryFile(entryFile)
	targetPath := filepath.Join(skillRoot, dirName, entryFile)
	resolvedTargetPath, err := resolveExistingRuleChainSkillPath(skillRoot, targetPath)
	if err != nil {
		return "", err
	}
	content, err := os.ReadFile(resolvedTargetPath)
	if err != nil {
		return "", err
	}
	trimmed := strings.TrimSpace(string(content))
	if trimmed == "" {
		return "", fmt.Errorf("%s 为空", entryFile)
	}
	return trimmed, nil
}

type ruleChainSkillPendingDeletion struct {
	originalPath string
	stagedPath   string
}

// PrepareDeleteRuleChainSkillDir 将规则链 Skill 目录原子搬迁到临时回收名，供后续 finalize/restore。
func PrepareDeleteRuleChainSkillDir(skillRoot string, meta RuleChainSkillMeta) (*ruleChainSkillPendingDeletion, error) {
	dirName := normalizeRuleChainSkillDirName(meta.DirName)
	if dirName == "" {
		return &ruleChainSkillPendingDeletion{}, nil
	}
	skillDirPath := filepath.Join(skillRoot, dirName)
	if !isWithinSkillRoot(skillRoot, skillDirPath) {
		return nil, fmt.Errorf("skill 目录路径越界")
	}
	if _, err := os.Lstat(skillDirPath); err != nil {
		if os.IsNotExist(err) {
			return &ruleChainSkillPendingDeletion{}, nil
		}
		return nil, err
	}
	recycleRoot := buildRuleChainSkillDeletionRecycleRoot(skillRoot)
	if err := os.MkdirAll(recycleRoot, 0o755); err != nil {
		return nil, err
	}
	pending := &ruleChainSkillPendingDeletion{
		originalPath: skillDirPath,
		stagedPath:   filepath.Join(recycleRoot, buildRuleChainSkillDeletionStagingName(dirName)),
	}
	if !isWithinSkillRoot(recycleRoot, pending.stagedPath) {
		return nil, fmt.Errorf("skill 临时目录路径越界")
	}
	if err := os.Rename(skillDirPath, pending.stagedPath); err != nil {
		if os.IsNotExist(err) {
			return &ruleChainSkillPendingDeletion{}, nil
		}
		return nil, err
	}
	return pending, nil
}

// DeleteRuleChainSkillDir 删除规则链 Skill 目录；目录不存在时视为成功。
func DeleteRuleChainSkillDir(skillRoot string, meta RuleChainSkillMeta) error {
	pending, err := PrepareDeleteRuleChainSkillDir(skillRoot, meta)
	if err != nil {
		return err
	}
	return pending.Finalize()
}

func (p *ruleChainSkillPendingDeletion) Finalize() error {
	if p == nil || p.stagedPath == "" {
		return nil
	}
	if _, err := os.Lstat(p.stagedPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.RemoveAll(p.stagedPath)
}

func (p *ruleChainSkillPendingDeletion) Restore() error {
	if p == nil || p.stagedPath == "" {
		return nil
	}
	if _, err := os.Lstat(p.stagedPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if _, err := os.Lstat(p.originalPath); err == nil {
		return fmt.Errorf("skill 原目录已存在，无法恢复")
	} else if !os.IsNotExist(err) {
		return err
	}
	return os.Rename(p.stagedPath, p.originalPath)
}

// ValidateGeneratedRuleChainSkillContent 校验生成的 Skill 至少包含最小关键锚点。
func ValidateGeneratedRuleChainSkillContent(content string, in RuleChainSkillPromptInput) error {
	missing := make([]string, 0, 4)
	for _, anchor := range BuildRuleChainSkillAcceptanceAnchors(in) {
		if !strings.Contains(content, anchor) {
			missing = append(missing, anchor)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("生成的 SKILL.md 缺少关键锚点: %s", strings.Join(missing, "; "))
	}
	return nil
}

// BuildRuleChainSkillSignature 构建稳定的规则链 Skill 签名。
func BuildRuleChainSkillSignature(description string, requestMetadataParams string, requestBodyParams string, responseBodyParams string) string {
	payload, _ := json.Marshal(struct {
		Description           string `json:"description"`
		RequestMetadataParams string `json:"request_metadata_params"`
		RequestBodyParams     string `json:"request_body_params"`
		ResponseBodyParams    string `json:"response_body_params"`
	}{
		Description:           strings.TrimSpace(description),
		RequestMetadataParams: canonicalizeRuleChainSkillJSON(requestMetadataParams),
		RequestBodyParams:     canonicalizeRuleChainSkillJSON(requestBodyParams),
		ResponseBodyParams:    canonicalizeRuleChainSkillJSON(responseBodyParams),
	})
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

// SanitizeRuleChainSkillDirName 规范化 Skill 目录名。
func SanitizeRuleChainSkillDirName(name string) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	normalized = ruleChainSkillDirNameSanitizer.ReplaceAllString(normalized, "-")
	normalized = strings.Trim(normalized, "-")
	if len(normalized) > maxRuleChainSkillDirNameLength {
		normalized = normalized[:maxRuleChainSkillDirNameLength]
		normalized = strings.Trim(normalized, "-")
	}
	return normalized
}

func normalizeRuleChainSkillDirName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" || filepath.IsAbs(trimmed) {
		return ""
	}
	if trimmed != filepath.Base(trimmed) || trimmed == "." || trimmed == ".." {
		return ""
	}
	return SanitizeRuleChainSkillDirName(trimmed)
}

func normalizeRuleChainSkillEntryFile(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" || filepath.IsAbs(trimmed) {
		return defaultRuleChainSkillEntryFile
	}
	if trimmed != filepath.Base(trimmed) || trimmed == "." || trimmed == ".." {
		return defaultRuleChainSkillEntryFile
	}
	if trimmed != defaultRuleChainSkillEntryFile {
		return defaultRuleChainSkillEntryFile
	}
	return defaultRuleChainSkillEntryFile
}

func isWithinSkillRoot(skillRoot string, targetPath string) bool {
	rootAbs, err := filepath.Abs(skillRoot)
	if err != nil {
		return false
	}
	targetAbs, err := filepath.Abs(targetPath)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func resolveExistingRuleChainSkillPath(skillRoot string, targetPath string) (string, error) {
	if !isWithinSkillRoot(skillRoot, targetPath) {
		return "", errRuleChainSkillPathEscape
	}
	rootAbs, err := filepath.Abs(skillRoot)
	if err != nil {
		return "", err
	}
	resolvedRoot, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		if os.IsNotExist(err) {
			resolvedRoot = rootAbs
		} else {
			return "", err
		}
	}
	resolvedTarget, err := filepath.EvalSymlinks(targetPath)
	if err != nil {
		return "", err
	}
	if !isWithinSkillRoot(resolvedRoot, resolvedTarget) {
		return "", errRuleChainSkillPathEscape
	}
	return resolvedTarget, nil
}

func buildRuleChainSkillDeletionStagingName(dirName string) string {
	return fmt.Sprintf(".%s.deleting.%d", dirName, time.Now().UnixNano())
}

func buildRuleChainSkillDeletionRecycleRoot(skillRoot string) string {
	absSkillRoot, err := filepath.Abs(skillRoot)
	if err != nil {
		return filepath.Join(filepath.Dir(filepath.Clean(skillRoot)), ".deleted-rulechain-skills")
	}
	return filepath.Join(filepath.Dir(absSkillRoot), ".deleted-rulechain-skills")
}

func canonicalizeRuleChainSkillJSON(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	var value interface{}
	if err := json.Unmarshal([]byte(trimmed), &value); err != nil {
		return trimmed
	}

	normalized, err := json.Marshal(value)
	if err != nil {
		return trimmed
	}
	return string(normalized)
}

func BuildRuleChainSkillSyncExecutePath(ruleChainID string) string {
	return fmt.Sprintf("/api/v1/rules/%s/execute/%s", strings.TrimSpace(ruleChainID), normalizeRuleChainSkillMsgType(""))
}

func BuildRuleChainSkillSyncExecutePathWithMsgType(ruleChainID string, msgType string) string {
	return fmt.Sprintf("/api/v1/rules/%s/execute/%s", strings.TrimSpace(ruleChainID), normalizeRuleChainSkillMsgType(msgType))
}

func BuildRuleChainSkillSyncExecutePathTemplate() string {
	return "/api/v1/rules/{id}/execute/{msgType}"
}

func BuildRuleChainSkillRequestBodyExample(in RuleChainSkillPromptInput) string {
	metadataKey := firstRuleChainSkillParamName(in.RequestMetadataParams, "tenant")
	metadataValue := firstRuleChainSkillExampleValue(in.RequestMetadataParams, "metadata")
	dataKey := firstRuleChainSkillParamName(in.RequestBodyParams, "input")
	dataValue := firstRuleChainSkillExampleValue(in.RequestBodyParams, "data")
	return fmt.Sprintf(`{"metadata": {"%s": "%s"}, "data": {"%s": "%s"}}`, metadataKey, metadataValue, dataKey, dataValue)
}

func BuildRuleChainSkillResponseReadHint(in RuleChainSkillPromptInput) string {
	field := firstRuleChainSkillParamName(in.ResponseBodyParams, "result")
	return fmt.Sprintf("返回体中的 data 可继续按业务字段读取，例如 response.data.%s", field)
}

func BuildRuleChainSkillResponseReadAnchor(in RuleChainSkillPromptInput) string {
	return "response_read:"
}

func BuildRuleChainSkillResultExplanationAnchor(in RuleChainSkillPromptInput) string {
	return "result_explanation:"
}

func BuildRuleChainSkillAcceptanceAnchors(in RuleChainSkillPromptInput) []string {
	return []string{
		BuildRuleChainSkillSignatureAnchor(BuildRuleChainSkillSignature(in.Description, in.RequestMetadataParams, in.RequestBodyParams, in.ResponseBodyParams)),
		fmt.Sprintf("rule_chain_id: %s", strings.TrimSpace(in.RuleChainID)),
		fmt.Sprintf("execute_path: %s", BuildRuleChainSkillSyncExecutePathWithMsgType(in.RuleChainID, in.MsgType)),
		fmt.Sprintf("request_body: %s", BuildRuleChainSkillRequestBodyExample(in)),
		BuildRuleChainSkillResultExplanationAnchor(in),
		BuildRuleChainSkillResponseReadAnchor(in),
		"metadata 和 data 必须分开整理",
	}
}

func BuildRuleChainSkillSignatureAnchor(signature string) string {
	return fmt.Sprintf("skill_signature: %s", strings.TrimSpace(signature))
}

func BuildRuleChainSkillBaseDirName(ruleChain *types.RuleChain) string {
	if dirName := SanitizeRuleChainSkillDirName(ruleChain.RuleChain.Name); dirName != "" {
		return dirName
	}
	if dirName := SanitizeRuleChainSkillDirName(ruleChain.RuleChain.ID); dirName != "" {
		return dirName
	}
	return "rulechain-skill"
}

func InferRuleChainSkillMsgType(ruleChain *types.RuleChain) string {
	if ruleChain == nil {
		return defaultRuleChainSkillMsgType
	}
	configuration := asJSONMap(ruleChain.RuleChain.Configuration)
	flowgram := asJSONMap(configuration["flowgram"])
	if msgType := normalizeRuleChainSkillMsgType(stringFromJSONValue(flowgram["entry_msg_type"])); msgType != defaultRuleChainSkillMsgType {
		return msgType
	}
	if msgType := normalizeRuleChainSkillMsgType(stringFromJSONValue(flowgram["entryMsgType"])); msgType != defaultRuleChainSkillMsgType {
		return msgType
	}
	for _, key := range []string{"entryMsgType", "msgType", "defaultMsgType", "notifyMsgType"} {
		if msgType := normalizeRuleChainSkillMsgType(stringFromJSONValue(ruleChain.RuleChain.AdditionalInfo[key])); msgType != defaultRuleChainSkillMsgType {
			return msgType
		}
	}
	if msgType := inferRuleChainSkillMsgTypeFromEndpoints(asJSONMap(ruleChain.Metadata)); msgType != "" {
		return msgType
	}
	return defaultRuleChainSkillMsgType
}

func normalizeRuleChainSkillMsgType(msgType string) string {
	trimmed := strings.TrimSpace(msgType)
	if trimmed == "" {
		return defaultRuleChainSkillMsgType
	}
	return trimmed
}

func inferRuleChainSkillMsgTypeFromEndpoints(metadata map[string]interface{}) string {
	endpoints, ok := metadata["endpoints"].([]interface{})
	if !ok || len(endpoints) == 0 {
		return ""
	}
	sorted := append([]interface{}(nil), endpoints...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return rankRuleChainSkillEndpointType(stringFromJSONValue(asJSONMap(sorted[i])["type"])) <
			rankRuleChainSkillEndpointType(stringFromJSONValue(asJSONMap(sorted[j])["type"]))
	})
	for _, endpoint := range sorted {
		ep := asJSONMap(endpoint)
		if stringFromJSONValue(ep["type"]) == "endpoint/schedule" {
			continue
		}
		routers, ok := ep["routers"].([]interface{})
		if !ok {
			continue
		}
		for _, router := range routers {
			path := stringFromJSONValue(asJSONMap(asJSONMap(router)["from"])["path"])
			if candidate := ruleChainSkillMsgTypeFromPath(path); candidate != "" {
				return candidate
			}
		}
	}
	return ""
}

func rankRuleChainSkillEndpointType(endpointType string) int {
	lower := strings.ToLower(strings.TrimSpace(endpointType))
	if strings.Contains(lower, "rest") || strings.Contains(lower, "http") {
		return 0
	}
	return 1
}

func ruleChainSkillMsgTypeFromPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" || ruleChainSkillLooksLikeCron(trimmed) {
		return ""
	}
	if strings.HasPrefix(trimmed, "/") {
		parts := strings.Split(strings.TrimRight(trimmed, "/"), "/")
		for i := len(parts) - 1; i >= 0; i-- {
			part := strings.TrimSpace(parts[i])
			if part == "" {
				continue
			}
			part = regexp.MustCompile(`\{[^}]+\}`).ReplaceAllString(part, "")
			part = strings.TrimSpace(part)
			if part != "" && !strings.Contains(part, "/") {
				return part
			}
		}
		return ""
	}
	parts := strings.Split(trimmed, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		part := strings.TrimSpace(parts[i])
		if part == "" || part == "+" || part == "#" || strings.Contains(part, "*") {
			continue
		}
		return part
	}
	if regexp.MustCompile(`^[a-zA-Z0-9_.\-]+$`).MatchString(trimmed) {
		return trimmed
	}
	return ""
}

func ruleChainSkillLooksLikeCron(path string) bool {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return false
	}
	if strings.Count(trimmed, " ") >= 4 {
		return true
	}
	return regexp.MustCompile(`^[\d\*\-\/,\s]+$`).MatchString(trimmed) && strings.Count(trimmed, " ") >= 1
}

func RuleChainSkillDirExists(skillRoot string, dirName string) (bool, error) {
	dirName = normalizeRuleChainSkillDirName(dirName)
	if dirName == "" {
		return false, nil
	}
	target := filepath.Join(skillRoot, dirName)
	if !isWithinSkillRoot(skillRoot, target) {
		return false, nil
	}
	_, err := os.Lstat(target)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func BuildRuleChainSkillConflictDirName(baseDir string, ruleChainID string) string {
	baseDir = normalizeRuleChainSkillDirName(baseDir)
	if baseDir == "" {
		baseDir = "rulechain-skill"
	}
	sum := sha256.Sum256([]byte(strings.TrimSpace(ruleChainID)))
	suffix := hex.EncodeToString(sum[:])[:8]
	maxBaseLen := maxRuleChainSkillDirNameLength - 1 - len(suffix)
	if maxBaseLen < 1 {
		maxBaseLen = 1
	}
	if len(baseDir) > maxBaseLen {
		baseDir = strings.Trim(baseDir[:maxBaseLen], "-")
		if baseDir == "" {
			baseDir = "rulechain-skill"
		}
	}
	return baseDir + "-" + suffix
}

func effectiveRuleChainSkillRootPath(root string) string {
	trimmed := strings.TrimSpace(root)
	if trimmed == "" {
		return "/app/skills"
	}
	return trimmed
}

func marshalRuleChainSkillJSON(v interface{}) string {
	if v == nil {
		return "[]"
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return canonicalizeRuleChainSkillJSON(string(b))
}

func stringFromJSONValue(v interface{}) string {
	if v == nil {
		return ""
	}
	switch typed := v.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func int64FromJSONValue(v interface{}) int64 {
	switch typed := v.(type) {
	case int:
		return int64(typed)
	case int32:
		return int64(typed)
	case int64:
		return typed
	case float64:
		return int64(typed)
	case json.Number:
		n, err := typed.Int64()
		if err == nil {
			return n
		}
	case string:
		var n int64
		if _, err := fmt.Sscan(strings.TrimSpace(typed), &n); err == nil {
			return n
		}
	}
	return 0
}

func firstRuleChainSkillParamName(raw string, fallback string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback
	}
	var arr []map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &arr); err == nil {
		for _, item := range arr {
			if name := stringFromJSONValue(item["name"]); name != "" {
				return name
			}
		}
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(trimmed), &obj); err == nil {
		if fields, ok := obj["fields"].([]interface{}); ok {
			for _, field := range fields {
				if fm, ok := field.(map[string]interface{}); ok {
					if name := stringFromJSONValue(fm["name"]); name != "" {
						return name
					}
				}
			}
		}
	}
	return fallback
}

func firstRuleChainSkillExampleValue(raw string, section string) string {
	name := strings.ToLower(firstRuleChainSkillParamName(raw, section))
	switch name {
	case "tenant":
		return "cn"
	case "city":
		return "Beijing"
	default:
		return "example"
	}
}
