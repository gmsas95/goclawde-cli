package persona

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ProposalStore manages evolution proposals
type ProposalStore struct {
	db *gorm.DB
}

// NewProposalStore creates a new proposal store
func NewProposalStore(db *gorm.DB) *ProposalStore {
	return &ProposalStore{db: db}
}

// CreateProposal creates a new proposal
func (ps *ProposalStore) CreateProposal(proposal *EvolutionProposal) error {
	if proposal.ID == "" {
		proposal.ID = fmt.Sprintf("proposal_%d", time.Now().UnixNano())
	}
	if proposal.Status == "" {
		proposal.Status = ProposalPending
	}
	proposal.CreatedAt = time.Now()
	proposal.UpdatedAt = time.Now()

	return ps.db.Create(proposal).Error
}

// GetProposal retrieves a proposal by ID
func (ps *ProposalStore) GetProposal(id string) (*EvolutionProposal, error) {
	var proposal EvolutionProposal
	err := ps.db.First(&proposal, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &proposal, nil
}

// ListProposals lists proposals with optional filters
func (ps *ProposalStore) ListProposals(status ProposalStatus, limit int) ([]*EvolutionProposal, error) {
	var proposals []*EvolutionProposal

	query := ps.db.Order("created_at DESC")

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&proposals).Error
	return proposals, err
}

// ListPendingProposals returns all pending proposals
func (ps *ProposalStore) ListPendingProposals() ([]*EvolutionProposal, error) {
	return ps.ListProposals(ProposalPending, 0)
}

// UpdateProposalStatus updates the status of a proposal
func (ps *ProposalStore) UpdateProposalStatus(id string, status ProposalStatus) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	switch status {
	case ProposalApproved:
		now := time.Now()
		updates["applied_at"] = now
	case ProposalRejected:
		now := time.Now()
		updates["rejected_at"] = now
	}

	return ps.db.Model(&EvolutionProposal{}).Where("id = ?", id).Updates(updates).Error
}

// SetRejectionReason sets the rejection reason for a proposal
func (ps *ProposalStore) SetRejectionReason(id string, reason string) error {
	return ps.db.Model(&EvolutionProposal{}).Where("id = ?", id).Updates(map[string]interface{}{
		"rejection_reason": reason,
		"updated_at":       time.Now(),
	}).Error
}

// DeleteProposal permanently deletes a proposal
func (ps *ProposalStore) DeleteProposal(id string) error {
	return ps.db.Delete(&EvolutionProposal{}, "id = ?", id).Error
}

// CountPendingProposals returns the number of pending proposals
func (ps *ProposalStore) CountPendingProposals() (int64, error) {
	var count int64
	err := ps.db.Model(&EvolutionProposal{}).Where("status = ?", ProposalPending).Count(&count).Error
	return count, err
}

// ProposalManager provides high-level proposal management
type ProposalManager struct {
	store *ProposalStore
}

// NewProposalManager creates a new proposal manager
func NewProposalManager(db *gorm.DB) *ProposalManager {
	return &ProposalManager{
		store: NewProposalStore(db),
	}
}

// Create creates a new proposal
func (pm *ProposalManager) Create(proposal *EvolutionProposal) error {
	return pm.store.CreateProposal(proposal)
}

// Get retrieves a proposal by ID
func (pm *ProposalManager) Get(id string) (*EvolutionProposal, error) {
	return pm.store.GetProposal(id)
}

// ListPending returns all pending proposals
func (pm *ProposalManager) ListPending() ([]*EvolutionProposal, error) {
	return pm.store.ListPendingProposals()
}

// ListAll returns all proposals
func (pm *ProposalManager) ListAll(limit int) ([]*EvolutionProposal, error) {
	return pm.store.ListProposals("", limit)
}

// Approve marks a proposal as approved
func (pm *ProposalManager) Approve(id string) error {
	return pm.store.UpdateProposalStatus(id, ProposalApproved)
}

// Reject marks a proposal as rejected with an optional reason
func (pm *ProposalManager) Reject(id string, reason string) error {
	if reason != "" {
		if err := pm.store.SetRejectionReason(id, reason); err != nil {
			return err
		}
	}
	return pm.store.UpdateProposalStatus(id, ProposalRejected)
}

// Apply marks a proposal as applied (should be called after applying changes)
func (pm *ProposalManager) Apply(id string) error {
	return pm.store.UpdateProposalStatus(id, ProposalApplied)
}

// GetPendingCount returns the number of pending proposals
func (pm *ProposalManager) GetPendingCount() (int64, error) {
	return pm.store.CountPendingProposals()
}

// HasPendingProposals returns true if there are pending proposals
func (pm *ProposalManager) HasPendingProposals() bool {
	count, err := pm.store.CountPendingProposals()
	return err == nil && count > 0
}

// DisplayProposal returns a formatted string representation of a proposal
func (pm *ProposalManager) DisplayProposal(proposal *EvolutionProposal) string {
	var result string

	result += fmt.Sprintf("Proposal: %s\n", proposal.ID)
	result += strings.Repeat("=", len("Proposal: "+proposal.ID)) + "\n\n"

	result += fmt.Sprintf("Type: %s\n", proposal.Type)
	result += fmt.Sprintf("Title: %s\n", proposal.Title)
	result += fmt.Sprintf("Confidence: %.0f%%\n", proposal.Confidence*100)
	result += fmt.Sprintf("Status: %s\n\n", proposal.Status)

	if proposal.Description != "" {
		result += fmt.Sprintf("Description:\n%s\n\n", proposal.Description)
	}

	if proposal.Rationale != "" {
		result += fmt.Sprintf("Rationale:\n%s\n\n", proposal.Rationale)
	}

	if proposal.Change != nil {
		result += "Proposed Change:\n"
		result += fmt.Sprintf("  Field: %s\n", proposal.Change.Field)
		result += fmt.Sprintf("  Operation: %s\n", proposal.Change.Operation)
		result += fmt.Sprintf("  Value: %v\n", proposal.Change.Value)
		if proposal.Change.OldValue != nil {
			result += fmt.Sprintf("  Previous: %v\n", proposal.Change.OldValue)
		}
		result += "\n"
	}

	result += fmt.Sprintf("Created: %s\n", proposal.CreatedAt.Format("2006-01-02 15:04"))

	if proposal.AppliedAt != nil {
		result += fmt.Sprintf("Applied: %s\n", proposal.AppliedAt.Format("2006-01-02 15:04"))
	}

	if proposal.RejectedAt != nil {
		result += fmt.Sprintf("Rejected: %s\n", proposal.RejectedAt.Format("2006-01-02 15:04"))
		if proposal.RejectionReason != "" {
			result += fmt.Sprintf("Reason: %s\n", proposal.RejectionReason)
		}
	}

	return result
}

// DisplayProposalsList returns a formatted list of proposals
func (pm *ProposalManager) DisplayProposalsList(proposals []*EvolutionProposal) string {
	if len(proposals) == 0 {
		return "No proposals found.\n"
	}

	var result string
	result += fmt.Sprintf("Found %d proposal(s):\n\n", len(proposals))

	for i, p := range proposals {
		statusIcon := "⏳"
		switch p.Status {
		case ProposalApproved:
			statusIcon = "✅"
		case ProposalRejected:
			statusIcon = "❌"
		case ProposalApplied:
			statusIcon = "✓"
		}

		result += fmt.Sprintf("%d. %s [%s] %s (%.0f%% confidence)\n",
			i+1, statusIcon, p.Type, p.Title, p.Confidence*100)
		result += fmt.Sprintf("   ID: %s\n", p.ID)
		result += fmt.Sprintf("   Created: %s\n", p.CreatedAt.Format("2006-01-02 15:04"))

		if i < len(proposals)-1 {
			result += "\n"
		}
	}

	return result
}

// AutoMigrateProposalTables runs migrations for proposal-related tables
func AutoMigrateProposalTables(db *gorm.DB) error {
	return db.AutoMigrate(
		&EvolutionProposal{},
		&DetectedPattern{},
		&SkillUsageRecord{},
		&PersonaVersion{},
		&Notification{},
	)
}
