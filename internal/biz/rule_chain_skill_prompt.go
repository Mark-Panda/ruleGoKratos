package biz

import (
	"fmt"
	"path"
	"strings"
)

// BuildRuleChainSkillGenerationPrompt 构建同步生成规则链 Skill 的指令。
func BuildRuleChainSkillGenerationPrompt(in RuleChainSkillPromptInput) string {
	outputFile := path.Join(effectiveRuleChainSkillRootPath(in.SkillRoot), in.DirName, defaultRuleChainSkillEntryFile)
	executePathTemplate := BuildRuleChainSkillSyncExecutePathTemplate()
	executePath := BuildRuleChainSkillSyncExecutePathWithMsgType(in.RuleChainID, in.MsgType)
	requestBodyExample := BuildRuleChainSkillRequestBodyExample(in)
	responseReadHint := BuildRuleChainSkillResponseReadHint(in)

	var b strings.Builder
	b.WriteString("你正在一个同步执行的托管 Agent Harness 中，为规则链生成 Agent-oriented Skill。\n\n")
	b.WriteString("必须遵守以下强约束：\n")
	b.WriteString("1. 你必须调用 `run_skill`，不得只输出说明文字。\n")
	b.WriteString("2. `run_skill` 的 `skill_name` 固定为 `skill-creator-0.1.0`。\n")
	b.WriteString(fmt.Sprintf("3. 输出文件固定为 `%s`，不得写入其他路径，也不得改文件名。\n", outputFile))
	b.WriteString("4. 生成的 Skill 面向 Agent，不要写面向终端用户、浏览器用户或人工操作员的使用说明。\n")
	b.WriteString("5. 生成内容必须是可执行的 SKILL.md，重点说明：任务目标、输入契约、输出契约、约束、失败处理、判定结果。\n\n")

	b.WriteString("规则链上下文：\n")
	b.WriteString(fmt.Sprintf("- rule_chain_id: %s\n", strings.TrimSpace(in.RuleChainID)))
	b.WriteString(fmt.Sprintf("- rule_chain_name: %s\n", strings.TrimSpace(in.RuleChainName)))
	b.WriteString(fmt.Sprintf("- description: %s\n", strings.TrimSpace(in.Description)))
	b.WriteString(fmt.Sprintf("- request_metadata_params: %s\n", strings.TrimSpace(in.RequestMetadataParams)))
	b.WriteString(fmt.Sprintf("- request_message_body_params: %s\n", strings.TrimSpace(in.RequestBodyParams)))
	b.WriteString(fmt.Sprintf("- response_message_body_params: %s\n\n", strings.TrimSpace(in.ResponseBodyParams)))
	b.WriteString("同步执行接口契约（必须写进 Skill）：\n")
	b.WriteString(fmt.Sprintf("- HTTP 路径形状：`POST %s`\n", executePathTemplate))
	b.WriteString(fmt.Sprintf("- 当前规则链示例路径：`POST %s`\n", executePath))
	b.WriteString(fmt.Sprintf("- 当前规则链推导的 msgType：`%s`\n", normalizeRuleChainSkillMsgType(in.MsgType)))
	b.WriteString(fmt.Sprintf("- 请求体必须显式区分 metadata/data：`%s`\n", requestBodyExample))
	b.WriteString(fmt.Sprintf("- 返回读取方式：%s\n\n", responseReadHint))

	b.WriteString("你写入的 Skill 必须明确包含以下内容：\n")
	b.WriteString("- metadata/data 整理规则：请求的 metadata 与 data 要分开描述；metadata 表示上下文与控制信息，data 表示主业务输入；若字段缺失，Skill 要说明如何保守降级。\n")
	b.WriteString("- 同步执行接口：说明调用方会通过同步 Harness 执行该 Skill，期望一次完成，不依赖异步轮询，也不要假设后续人工补救步骤。\n")
	b.WriteString("- 结果解释：成功时应该产出什么结构、如何判断结果可用；失败时如何返回可读错误，避免伪造成功。\n")
	b.WriteString("- 失败兜底：当 metadata/data 缺字段、规则链描述不足、输出结构不稳定时，要优先给出安全、明确、可恢复的失败结果，而不是编造答案。\n\n")
	b.WriteString("为了便于服务端验收，你输出的 SKILL.md 里必须原样包含以下关键锚点：\n")
	for _, anchor := range BuildRuleChainSkillAcceptanceAnchors(in) {
		b.WriteString(fmt.Sprintf("- `%s`\n", anchor))
	}
	b.WriteString("\n")

	b.WriteString("执行步骤要求：\n")
	b.WriteString("1. 先根据上述规则链信息构造 Skill 内容。\n")
	b.WriteString("2. 调用 `run_skill`，让 `skill-creator-0.1.0` 把最终内容写入指定文件。\n")
	b.WriteString("3. 如果工具返回失败、或你无法保证写入内容满足约束，请直接返回失败原因，不要声称成功。\n")
	b.WriteString("4. 如果工具成功，请简短说明已生成 Skill，并指出目标文件路径。\n")

	return b.String()
}
