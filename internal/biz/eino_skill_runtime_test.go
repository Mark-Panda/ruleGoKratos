package biz

import (
	"context"
	"errors"
	"testing"

	einoskill "github.com/cloudwego/eino/adk/middlewares/skill"
)

func TestParseTriggerSkip(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantClean      string
		wantTrigger    string
		wantSkip       string
	}{
		{
			name:           "empty description",
			input:          "",
			wantClean:      "",
			wantTrigger:    "",
			wantSkip:       "",
		},
		{
			name:           "description without trigger/skip",
			input:          "This is a simple description.",
			wantClean:      "This is a simple description.",
			wantTrigger:    "",
			wantSkip:       "",
		},
		{
			name: "description with TRIGGER when",
			input: `Use this skill for PDF generation.
TRIGGER when: user asks to create a PDF
SKIP when: user needs Word document`,
			wantClean: "Use this skill for PDF generation.",
			wantTrigger: "user asks to create a PDF",
			wantSkip:    "user needs Word document",
		},
		{
			name: "description with TRIGGER: (colon only)",
			input: `Use this skill for PDF.
TRIGGER: user wants PDF
SKIP: user wants Word`,
			wantClean:   "Use this skill for PDF.",
			wantTrigger: "user wants PDF",
			wantSkip:    "user wants Word",
		},
		{
			name:           "only trigger",
			input:          "Use this.\nTRIGGER when: pdf",
			wantClean:      "Use this.",
			wantTrigger:    "pdf",
			wantSkip:       "",
		},
		{
			name:           "only skip",
			input:          "Use this.\nSKIP when: word",
			wantClean:      "Use this.",
			wantTrigger:    "",
			wantSkip:       "word",
		},
		{
			name: "description with long description",
			input: `Professional PDF document generation with beautiful design.
TRIGGER when: user wants to create a PDF document
SKIP when: user needs a Word document`,
			wantClean:   "Professional PDF document generation with beautiful design.",
			wantTrigger: "user wants to create a PDF document",
			wantSkip:    "user needs a Word document",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clean, trigger, skip := parseTriggerSkip(tt.input)
			if clean != tt.wantClean {
				t.Errorf("parseTriggerSkip clean = %q, want %q", clean, tt.wantClean)
			}
			if trigger != tt.wantTrigger {
				t.Errorf("parseTriggerSkip trigger = %q, want %q", trigger, tt.wantTrigger)
			}
			if skip != tt.wantSkip {
				t.Errorf("parseTriggerSkip skip = %q, want %q", skip, tt.wantSkip)
			}
		})
	}
}

func TestSuggestClosestSkillName(t *testing.T) {
	skills := []einoskill.FrontMatter{
		{Name: "minimax-pdf"},
		{Name: "minimax-docx"},
		{Name: "minimax-xlsx"},
		{Name: "pptx-generator"},
		{Name: "tavily"},
		{Name: "skill-creator"},
	}

	tests := []struct {
		name      string
		requested string
		want      string
	}{
		{
			name:      "exact match",
			requested: "minimax-pdf",
			want:      "minimax-pdf",
		},
		{
			name:      "suffix match pdf",
			requested: "pdf",
			want:      "minimax-pdf",
		},
		{
			name:      "suffix match docx",
			requested: "docx",
			want:      "minimax-docx",
		},
		{
			name:      "suffix match xlsx",
			requested: "xlsx",
			want:      "minimax-xlsx",
		},
		{
			name:      "prefix match tavily",
			requested: "tavily",
			want:      "tavily",
		},
		{
			name:      "contains match pptx",
			requested: "pptx",
			want:      "pptx-generator",
		},
		{
			name:      "typo - missing char",
			requested: "minimax-pd",
			want:      "minimax-pdf",
		},
		{
			name:      "typo - wrong char",
			requested: "minimax-pdx",
			want:      "minimax-pdf",
		},
		{
			name:      "case insensitive",
			requested: "MINIMAX-PDF",
			want:      "minimax-pdf",
		},
		{
			name:      "no close match - random string",
			requested: "zzzzzzzz",
			want:      "",
		},
		{
			name:      "empty requested",
			requested: "",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := suggestClosestSkillName(skills, tt.requested)
			if got != tt.want {
				t.Errorf("suggestClosestSkillName(%q) = %q, want %q", tt.requested, got, tt.want)
			}
		})
	}
}

func TestSuggestClosestSkillNameEmptyList(t *testing.T) {
	got := suggestClosestSkillName([]einoskill.FrontMatter{}, "pdf")
	if got != "" {
		t.Errorf("suggestClosestSkillName with empty list = %q, want empty", got)
	}
}

func TestRecoverableSkillError(t *testing.T) {
	err := &recoverableSkillError{
		msg:        "skill not found: test",
		suggestion: "minimax-pdf",
	}

	if !err.IsRecoverable() {
		t.Error("recoverableSkillError should be recoverable")
	}

	if err.Error() != "skill not found: test" {
		t.Errorf("Error() = %q, want %q", err.Error(), "skill not found: test")
	}
}

func TestIsRecoverableToolError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "recoverable skill error",
			err:  &recoverableSkillError{msg: "skill not found"},
			want: true,
		},
		{
			name: "standard error",
			err:  errors.New("some error"),
			want: false,
		},
		{
			name: "wrapped recoverable error",
			err:  errors.New("wrapped: " + (&recoverableSkillError{msg: "skill not found"}).Error()),
			want: false, // errors.As should unwrap, but our IsRecoverable checks As internally
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRecoverableToolError(tt.err)
			if got != tt.want {
				t.Errorf("IsRecoverableToolError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestCustomSkillToolDescription(t *testing.T) {
	ctx := context.Background()
	skills := []einoskill.FrontMatter{
		{
			Name:        "minimax-pdf",
			Description: "PDF generation.\nTRIGGER when: user wants PDF\nSKIP when: user wants Word",
		},
		{
			Name:        "minimax-docx",
			Description: "Word document.\nTRIGGER when: user wants docx",
		},
		{
			Name:        "simple-skill",
			Description: "Simple description without trigger/skip",
		},
	}

	desc := customSkillToolDescription(ctx, skills)

	// Check that skill names are present
	if !contains(desc, "<name>\nminimax-pdf\n</name>") {
		t.Error("description should contain minimax-pdf name tag")
	}
	if !contains(desc, "<name>\nminimax-docx\n</name>") {
		t.Error("description should contain minimax-docx name tag")
	}

	// Check that trigger tags are present for skills with triggers
	if !contains(desc, "<trigger>\nuser wants PDF\n</trigger>") {
		t.Error("description should contain trigger tag for minimax-pdf")
	}
	if !contains(desc, "<skip>\nuser wants Word\n</skip>") {
		t.Error("description should contain skip tag for minimax-pdf")
	}

	// Check that there are 2 trigger tags (one for each skill with trigger)
	triggerCount := countOccurrences(desc, "<trigger>")
	if triggerCount != 2 {
		t.Errorf("should have 2 trigger tags (minimax-pdf and minimax-docx), got %d", triggerCount)
	}

	// Check that simple skill has no trigger/skip tags
	if contains(desc, "<trigger>\nSimple") {
		t.Error("simple-skill should not have trigger tag")
	}
}

func TestCustomSkillToolDescriptionEmpty(t *testing.T) {
	ctx := context.Background()
	desc := customSkillToolDescription(ctx, []einoskill.FrontMatter{})

	if !contains(desc, "(No skills available)") {
		t.Error("empty skills should show no skills available message")
	}
}

func TestCustomBuildContent(t *testing.T) {
	ctx := context.Background()
	skill := einoskill.Skill{
		FrontMatter: einoskill.FrontMatter{
			Name:        "test-skill",
			Description: "Test skill description",
		},
		Content:      "Skill body content",
		BaseDirectory: "/path/to/skill",
	}

	content, err := customBuildContent(ctx, skill, "{}")
	if err != nil {
		t.Fatalf("customBuildContent returned error: %v", err)
	}

	// Check wrapper is present
	if !contains(content, "<skill_instruction name=\"test-skill\" enforcement=\"mandatory\">") {
		t.Error("content should contain skill_instruction wrapper")
	}

	// Check base content is present
	if !contains(content, "正在启动 Skill：test-skill") {
		t.Error("content should contain launch message")
	}
	if !contains(content, "/path/to/skill") {
		t.Error("content should contain base directory")
	}
	if !contains(content, "Skill body content") {
		t.Error("content should contain skill content")
	}

	// Check compliance directive
	if !contains(content, "<compliance_directive>") {
		t.Error("content should contain compliance directive")
	}
	if !contains(content, "禁止行为") {
		t.Error("content should contain prohibition rules")
	}
}

func TestCustomSystemPrompt(t *testing.T) {
	ctx := context.Background()
	prompt := customSkillSystemPrompt(ctx, "skill")

	// Check key rules are present
	if !contains(prompt, "只调用列出的 Skill") {
		t.Error("system prompt should contain skill invocation rule")
	}
	if !contains(prompt, "精确匹配名称") {
		t.Error("system prompt should contain name matching rule")
	}
	if !contains(prompt, "触发条件检查") {
		t.Error("system prompt should contain trigger check rule")
	}
	if !contains(prompt, "跳过条件检查") {
		t.Error("system prompt should contain skip check rule")
	}
	if !contains(prompt, "错误自我纠正") {
		t.Error("system prompt should contain error self-correction rule")
	}
	if !contains(prompt, "严格执行") {
		t.Error("system prompt should contain strict execution rule")
	}

	// Check the tool name is embedded
	if !contains(prompt, "'skill' 工具") {
		t.Error("system prompt should reference the skill tool name")
	}
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func countOccurrences(s, substr string) int {
	count := 0
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			count++
		}
	}
	return count
}
