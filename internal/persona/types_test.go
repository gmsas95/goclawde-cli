package persona

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPatternThreshold(t *testing.T) {
	threshold := DefaultPatternThreshold()

	assert.Equal(t, 20, threshold.MinFrequency)
	assert.Equal(t, 0.7, threshold.MinConfidence)
	assert.Equal(t, 50, threshold.SkillUsageThreshold)
	assert.Equal(t, 30, threshold.TimeWindowDays)
	assert.Equal(t, 0.95, threshold.AutoApplyThreshold)
}

func TestEvolutionProposal(t *testing.T) {
	proposal := &EvolutionProposal{
		ID:         "test_proposal",
		Type:       ExpertiseProposal,
		Title:      "Add Go expertise",
		Confidence: 0.85,
		Status:     ProposalPending,
	}

	assert.True(t, proposal.IsHighConfidence(0.8))
	assert.False(t, proposal.IsHighConfidence(0.9))
	assert.Equal(t, "Add expertise: Add Go expertise", proposal.Summary())
}

func TestProposalStatus(t *testing.T) {
	assert.Equal(t, ProposalStatus("pending"), ProposalPending)
	assert.Equal(t, ProposalStatus("approved"), ProposalApproved)
	assert.Equal(t, ProposalStatus("rejected"), ProposalRejected)
	assert.Equal(t, ProposalStatus("applied"), ProposalApplied)
}

func TestPatternType(t *testing.T) {
	assert.Equal(t, PatternType("frequency"), FrequencyPattern)
	assert.Equal(t, PatternType("temporal"), TemporalPattern)
	assert.Equal(t, PatternType("skill_usage"), SkillUsagePattern)
	assert.Equal(t, PatternType("context_switch"), ContextSwitchPattern)
}

func TestProposalType(t *testing.T) {
	assert.Equal(t, ProposalType("expertise"), ExpertiseProposal)
	assert.Equal(t, ProposalType("preference"), PreferenceProposal)
	assert.Equal(t, ProposalType("value"), ValueProposal)
	assert.Equal(t, ProposalType("voice"), VoiceProposal)
	assert.Equal(t, ProposalType("goal"), GoalProposal)
}

func TestChangeType(t *testing.T) {
	assert.Equal(t, ChangeType("manual"), ChangeManual)
	assert.Equal(t, ChangeType("proposal"), ChangeProposal)
	assert.Equal(t, ChangeType("rollback"), ChangeRollback)
	assert.Equal(t, ChangeType("auto"), ChangeAuto)
}

func TestEvolutionConfig(t *testing.T) {
	config := DefaultEvolutionConfig()

	assert.True(t, config.Enabled)
	assert.True(t, config.Notifications)
	assert.False(t, config.AutoApplyHighConfidence)
	assert.Equal(t, 5, config.WeeklyAnalysis.MaxProposals)
}

func TestCanAutoApply(t *testing.T) {
	config := DefaultEvolutionConfig()
	config.AutoApplyHighConfidence = true

	highConfProposal := &EvolutionProposal{
		Confidence: 0.96,
	}
	lowConfProposal := &EvolutionProposal{
		Confidence: 0.85,
	}

	assert.True(t, highConfProposal.CanAutoApply(config))
	assert.False(t, lowConfProposal.CanAutoApply(config))

	// When auto-apply is disabled
	config.AutoApplyHighConfidence = false
	assert.False(t, highConfProposal.CanAutoApply(config))
}

func TestDetectedPattern(t *testing.T) {
	pattern := &DetectedPattern{
		ID:         "pattern_123",
		Type:       FrequencyPattern,
		Subject:    "golang",
		Frequency:  25,
		Confidence: 0.85,
	}

	assert.Equal(t, "pattern_123", pattern.ID)
	assert.Equal(t, FrequencyPattern, pattern.Type)
	assert.Equal(t, "golang", pattern.Subject)
	assert.Equal(t, 25, pattern.Frequency)
	assert.Equal(t, 0.85, pattern.Confidence)
}

func TestPersonaChange(t *testing.T) {
	change := &PersonaChange{
		Field:     "identity.expertise",
		Operation: "add",
		Value:     "Go",
	}

	assert.Equal(t, "identity.expertise", change.Field)
	assert.Equal(t, "add", change.Operation)
	assert.Equal(t, "Go", change.Value)
}

func TestWeeklyAnalysisConfig(t *testing.T) {
	config := DefaultWeeklyAnalysisConfig()

	assert.True(t, config.Enabled)
	assert.Equal(t, 0, config.DayOfWeek) // Sunday
	assert.Equal(t, 9, config.Hour)
	assert.Equal(t, 0, config.Minute)
	assert.Equal(t, 5, config.MaxProposals)
}
