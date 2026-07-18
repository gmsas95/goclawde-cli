package runtime

import (
	"testing"

	"github.com/gmsas95/myrai-cli/internal/llm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func testToolCall(name, args string) *llm.ToolCall {
	tc := &llm.ToolCall{}
	tc.Function.Name = name
	tc.Function.Arguments = args
	return tc
}

func TestValidateToolPolicy_PathOutsideWorkspaceBlocked(t *testing.T) {
	db := setupTestDB(t)
	store := NewStore(db)
	runner := NewRunner(RunnerConfig{
		Logger:        zap.NewNop(),
		Store:         store,
		WorkspaceRoot: t.TempDir(),
	})

	err := runner.policyEngine.ValidateToolCall(testToolCall("write_file", `{"path":"../../etc/passwd","content":"oops"}`))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "path policy violation")
}

func TestValidateToolPolicy_CommandPolicyBlocked(t *testing.T) {
	db := setupTestDB(t)
	store := NewStore(db)
	runner := NewRunner(RunnerConfig{
		Logger:        zap.NewNop(),
		Store:         store,
		WorkspaceRoot: t.TempDir(),
		AllowedCmds:   []string{"git", "go"},
	})

	err := runner.policyEngine.ValidateToolCall(testToolCall("exec_command", `{"command":"git status && rm -rf /"}`))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "shell control operators")
}

func TestValidateToolPolicy_AllowedCommandWithinWorkspace(t *testing.T) {
	workspace := t.TempDir()
	db := setupTestDB(t)
	store := NewStore(db)
	runner := NewRunner(RunnerConfig{
		Logger:        zap.NewNop(),
		Store:         store,
		WorkspaceRoot: workspace,
		AllowedCmds:   []string{"git", "go"},
	})

	err := runner.policyEngine.ValidateToolCall(testToolCall("exec_command", `{"command":"git status","path":"."}`))

	assert.NoError(t, err)
}
