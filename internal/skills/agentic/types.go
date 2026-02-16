package agentic

import (
	"github.com/gmsas95/myrai-cli/internal/skills"
)

type AgenticSkill struct {
	*skills.BaseSkill
	workspaceRoot string
}

func NewAgenticSkill(workspaceRoot string) *AgenticSkill {
	s := &AgenticSkill{
		BaseSkill:     skills.NewBaseSkill("agentic", "Advanced agentic capabilities for system and code analysis", "1.0.0"),
		workspaceRoot: workspaceRoot,
	}
	s.registerTools()
	return s
}
