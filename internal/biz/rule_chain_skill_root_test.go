package biz

import (
	"os"
	"testing"

	"ruleGoKratos/internal/conf"
)

func TestResolveRuleChainSkillRootDefaultsToWorkflowRoot(t *testing.T) {
	t.Setenv("RULE_CHAIN_SKILL_DIR", "")
	got := resolveRuleChainSkillRoot(nil)
	if got != "/workflow/skills" {
		t.Fatalf("expected default workflow skill root, got %q", got)
	}
}

func TestResolveRuleChainSkillRootUsesEnvOverride(t *testing.T) {
	t.Setenv("RULE_CHAIN_SKILL_DIR", "/tmp/workflow-skills")
	got := resolveRuleChainSkillRoot(nil)
	if got != "/tmp/workflow-skills" {
		t.Fatalf("expected env workflow skill root, got %q", got)
	}
}

func TestResolveRuleChainSkillRootFallsBackToConfig(t *testing.T) {
	_ = os.Unsetenv("RULE_CHAIN_SKILL_DIR")
	cfg := &conf.Bootstrap{
		Agent: &conf.Agent{
			Skill: &conf.Agent_Skill{
				Dir: "/custom/skill-root",
			},
		},
	}
	got := resolveRuleChainSkillRoot(cfg)
	if got != "/custom/skill-root" {
		t.Fatalf("expected config skill root, got %q", got)
	}
}
