package persona

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// VersionStore manages persona version history
type VersionStore struct {
	db *gorm.DB
}

// NewVersionStore creates a new version store
func NewVersionStore(db *gorm.DB) *VersionStore {
	return &VersionStore{db: db}
}

// SaveVersion saves a new version snapshot
func (vs *VersionStore) SaveVersion(version *PersonaVersion) error {
	if version.ID == "" {
		version.ID = fmt.Sprintf("version_%d", time.Now().UnixNano())
	}
	if version.Timestamp.IsZero() {
		version.Timestamp = time.Now()
	}

	return vs.db.Create(version).Error
}

// GetVersion retrieves a version by ID
func (vs *VersionStore) GetVersion(id string) (*PersonaVersion, error) {
	var version PersonaVersion
	err := vs.db.First(&version, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}

// ListVersions lists all versions ordered by timestamp (newest first)
func (vs *VersionStore) ListVersions(limit int) ([]*PersonaVersion, error) {
	var versions []*PersonaVersion

	query := vs.db.Order("timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&versions).Error
	return versions, err
}

// GetLatestVersion retrieves the most recent version
func (vs *VersionStore) GetLatestVersion() (*PersonaVersion, error) {
	var version PersonaVersion
	err := vs.db.Order("timestamp DESC").First(&version).Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}

// DeleteVersion deletes a specific version
func (vs *VersionStore) DeleteVersion(id string) error {
	return vs.db.Delete(&PersonaVersion{}, "id = ?", id).Error
}

// VersionManager provides high-level version management
type VersionManager struct {
	store         *VersionStore
	workspacePath string
}

// NewVersionManager creates a new version manager
func NewVersionManager(db *gorm.DB, workspacePath string) *VersionManager {
	return &VersionManager{
		store:         NewVersionStore(db),
		workspacePath: workspacePath,
	}
}

// SaveVersion creates a snapshot of the current persona state
func (vm *VersionManager) SaveVersion(identity *Identity, userProfile *UserProfile, changeType ChangeType, description string, proposalID *string) (*PersonaVersion, error) {
	// Get the previous version ID if any
	var prevVersionID *string
	if latest, err := vm.store.GetLatestVersion(); err == nil {
		prevVersionID = &latest.ID
	}

	// Deep copy identity and user profile
	identityCopy := vm.copyIdentity(identity)
	userProfileCopy := vm.copyUserProfile(userProfile)

	version := &PersonaVersion{
		ID:                fmt.Sprintf("version_%d", time.Now().UnixNano()),
		Timestamp:         time.Now(),
		Identity:          identityCopy,
		UserProfile:       userProfileCopy,
		ChangeType:        changeType,
		ChangeDescription: description,
		ProposalID:        proposalID,
		PreviousVersion:   prevVersionID,
	}

	if err := vm.store.SaveVersion(version); err != nil {
		return nil, err
	}

	return version, nil
}

// SaveManualVersion saves a version for manual edits
func (vm *VersionManager) SaveManualVersion(identity *Identity, userProfile *UserProfile, description string) (*PersonaVersion, error) {
	if description == "" {
		description = "Manual edit"
	}
	return vm.SaveVersion(identity, userProfile, ChangeManual, description, nil)
}

// SaveProposalVersion saves a version for an applied proposal
func (vm *VersionManager) SaveProposalVersion(identity *Identity, userProfile *UserProfile, proposal *EvolutionProposal) (*PersonaVersion, error) {
	description := fmt.Sprintf("Applied proposal: %s", proposal.Title)
	return vm.SaveVersion(identity, userProfile, ChangeProposal, description, &proposal.ID)
}

// ListVersions returns version history
func (vm *VersionManager) ListVersions(limit int) ([]*PersonaVersion, error) {
	return vm.store.ListVersions(limit)
}

// GetVersion retrieves a specific version
func (vm *VersionManager) GetVersion(id string) (*PersonaVersion, error) {
	return vm.store.GetVersion(id)
}

// Rollback restores a previous version
func (vm *VersionManager) Rollback(versionID string) (*PersonaVersion, *Identity, *UserProfile, error) {
	// Get the version to rollback to
	targetVersion, err := vm.store.GetVersion(versionID)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("version not found: %w", err)
	}

	if targetVersion.Identity == nil || targetVersion.UserProfile == nil {
		return nil, nil, nil, fmt.Errorf("version %s is incomplete", versionID)
	}

	// Create a new version entry for the rollback
	rollbackVersion, err := vm.SaveVersion(
		targetVersion.Identity,
		targetVersion.UserProfile,
		ChangeRollback,
		fmt.Sprintf("Rollback to version %s", versionID),
		nil,
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to save rollback version: %w", err)
	}

	return rollbackVersion, targetVersion.Identity, targetVersion.UserProfile, nil
}

// GetVersionDiff shows what changed between two versions
func (vm *VersionManager) GetVersionDiff(fromVersionID, toVersionID string) (*VersionDiff, error) {
	fromVersion, err := vm.store.GetVersion(fromVersionID)
	if err != nil {
		return nil, fmt.Errorf("from version not found: %w", err)
	}

	toVersion, err := vm.store.GetVersion(toVersionID)
	if err != nil {
		return nil, fmt.Errorf("to version not found: %w", err)
	}

	diff := &VersionDiff{
		FromVersion: fromVersion.ID,
		ToVersion:   toVersion.ID,
		FromTime:    fromVersion.Timestamp,
		ToTime:      toVersion.Timestamp,
	}

	// Compare identity
	if fromVersion.Identity != nil && toVersion.Identity != nil {
		diff.IdentityChanges = vm.compareIdentity(fromVersion.Identity, toVersion.Identity)
	}

	// Compare user profile
	if fromVersion.UserProfile != nil && toVersion.UserProfile != nil {
		diff.UserProfileChanges = vm.compareUserProfile(fromVersion.UserProfile, toVersion.UserProfile)
	}

	return diff, nil
}

// GetChangelog returns a summary of changes over time
func (vm *VersionManager) GetChangelog(limit int) ([]ChangelogEntry, error) {
	versions, err := vm.store.ListVersions(limit + 1)
	if err != nil {
		return nil, err
	}

	if len(versions) < 2 {
		return []ChangelogEntry{}, nil
	}

	var entries []ChangelogEntry
	for i := 0; i < len(versions)-1 && i < limit; i++ {
		current := versions[i]
		previous := versions[i+1]

		entry := ChangelogEntry{
			VersionID:   current.ID,
			Timestamp:   current.Timestamp,
			ChangeType:  current.ChangeType,
			Description: current.ChangeDescription,
		}

		// Calculate changes
		if current.Identity != nil && previous.Identity != nil {
			entry.IdentityChanges = len(vm.compareIdentity(previous.Identity, current.Identity))
		}
		if current.UserProfile != nil && previous.UserProfile != nil {
			entry.UserProfileChanges = len(vm.compareUserProfile(previous.UserProfile, current.UserProfile))
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// CleanupOldVersions removes versions older than the retention period
func (vm *VersionManager) CleanupOldVersions(retentionDays int) (int, error) {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	var versions []*PersonaVersion
	err := vm.store.db.Where("timestamp < ?", cutoff).Find(&versions).Error
	if err != nil {
		return 0, err
	}

	deleted := 0
	for _, v := range versions {
		// Keep at least 10 versions regardless of age
		if deleted >= len(versions)-10 {
			break
		}

		if err := vm.store.DeleteVersion(v.ID); err == nil {
			deleted++
		}
	}

	return deleted, nil
}

// copyIdentity creates a deep copy of an Identity
func (vm *VersionManager) copyIdentity(identity *Identity) *Identity {
	if identity == nil {
		return nil
	}

	idCopy := &Identity{
		Name:        identity.Name,
		Personality: identity.Personality,
		Voice:       identity.Voice,
	}

	if identity.Values != nil {
		idCopy.Values = make([]string, len(identity.Values))
		copy(idCopy.Values, identity.Values)
	}

	if identity.Expertise != nil {
		idCopy.Expertise = make([]string, len(identity.Expertise))
		copy(idCopy.Expertise, identity.Expertise)
	}

	return idCopy
}

// copyUserProfile creates a deep copy of a UserProfile
func (vm *VersionManager) copyUserProfile(profile *UserProfile) *UserProfile {
	if profile == nil {
		return nil
	}

	upCopy := &UserProfile{
		Name:               profile.Name,
		CommunicationStyle: profile.CommunicationStyle,
		CreatedAt:          profile.CreatedAt,
		UpdatedAt:          profile.UpdatedAt,
	}

	if profile.Preferences != nil {
		upCopy.Preferences = make(map[string]string)
		for k, v := range profile.Preferences {
			upCopy.Preferences[k] = v
		}
	}

	if profile.Expertise != nil {
		upCopy.Expertise = make([]string, len(profile.Expertise))
		copy(upCopy.Expertise, profile.Expertise)
	}

	if profile.Goals != nil {
		upCopy.Goals = make([]string, len(profile.Goals))
		copy(upCopy.Goals, profile.Goals)
	}

	return upCopy
}

// compareIdentity compares two identities and returns the differences
func (vm *VersionManager) compareIdentity(old, new *Identity) []Change {
	var changes []Change

	if old.Name != new.Name {
		changes = append(changes, Change{
			Field:    "identity.name",
			OldValue: old.Name,
			NewValue: new.Name,
		})
	}

	if old.Personality != new.Personality {
		changes = append(changes, Change{
			Field:    "identity.personality",
			OldValue: old.Personality,
			NewValue: new.Personality,
		})
	}

	if old.Voice != new.Voice {
		changes = append(changes, Change{
			Field:    "identity.voice",
			OldValue: old.Voice,
			NewValue: new.Voice,
		})
	}

	// Compare expertise
	expertiseChanges := vm.compareStringSlices(old.Expertise, new.Expertise)
	for _, c := range expertiseChanges {
		c.Field = "identity.expertise"
		changes = append(changes, c)
	}

	// Compare values
	valuesChanges := vm.compareStringSlices(old.Values, new.Values)
	for _, c := range valuesChanges {
		c.Field = "identity.values"
		changes = append(changes, c)
	}

	return changes
}

// compareUserProfile compares two user profiles and returns the differences
func (vm *VersionManager) compareUserProfile(old, new *UserProfile) []Change {
	var changes []Change

	if old.Name != new.Name {
		changes = append(changes, Change{
			Field:    "user.name",
			OldValue: old.Name,
			NewValue: new.Name,
		})
	}

	if old.CommunicationStyle != new.CommunicationStyle {
		changes = append(changes, Change{
			Field:    "user.communication_style",
			OldValue: old.CommunicationStyle,
			NewValue: new.CommunicationStyle,
		})
	}

	// Compare expertise
	expertiseChanges := vm.compareStringSlices(old.Expertise, new.Expertise)
	for _, c := range expertiseChanges {
		c.Field = "user.expertise"
		changes = append(changes, c)
	}

	// Compare goals
	goalsChanges := vm.compareStringSlices(old.Goals, new.Goals)
	for _, c := range goalsChanges {
		c.Field = "user.goals"
		changes = append(changes, c)
	}

	// Compare preferences
	for key, oldVal := range old.Preferences {
		if newVal, exists := new.Preferences[key]; exists {
			if oldVal != newVal {
				changes = append(changes, Change{
					Field:    fmt.Sprintf("user.preferences.%s", key),
					OldValue: oldVal,
					NewValue: newVal,
				})
			}
		} else {
			changes = append(changes, Change{
				Field:     fmt.Sprintf("user.preferences.%s", key),
				OldValue:  oldVal,
				NewValue:  nil,
				Operation: "removed",
			})
		}
	}

	for key, newVal := range new.Preferences {
		if _, exists := old.Preferences[key]; !exists {
			changes = append(changes, Change{
				Field:     fmt.Sprintf("user.preferences.%s", key),
				OldValue:  nil,
				NewValue:  newVal,
				Operation: "added",
			})
		}
	}

	return changes
}

// compareStringSlices compares two string slices
func (vm *VersionManager) compareStringSlices(old, new []string) []Change {
	var changes []Change

	// Find removed items
	for _, oldItem := range old {
		found := false
		for _, newItem := range new {
			if oldItem == newItem {
				found = true
				break
			}
		}
		if !found {
			changes = append(changes, Change{
				Field:     "",
				OldValue:  oldItem,
				NewValue:  nil,
				Operation: "removed",
			})
		}
	}

	// Find added items
	for _, newItem := range new {
		found := false
		for _, oldItem := range old {
			if newItem == oldItem {
				found = true
				break
			}
		}
		if !found {
			changes = append(changes, Change{
				Field:     "",
				OldValue:  nil,
				NewValue:  newItem,
				Operation: "added",
			})
		}
	}

	return changes
}

// Change represents a single change between two versions
type Change struct {
	Field     string      `json:"field"`
	OldValue  interface{} `json:"old_value"`
	NewValue  interface{} `json:"new_value"`
	Operation string      `json:"operation,omitempty"` // "added", "removed", "modified"
}

// VersionDiff represents the differences between two versions
type VersionDiff struct {
	FromVersion        string    `json:"from_version"`
	ToVersion          string    `json:"to_version"`
	FromTime           time.Time `json:"from_time"`
	ToTime             time.Time `json:"to_time"`
	IdentityChanges    []Change  `json:"identity_changes"`
	UserProfileChanges []Change  `json:"user_profile_changes"`
}

// ChangelogEntry represents a single changelog entry
type ChangelogEntry struct {
	VersionID          string     `json:"version_id"`
	Timestamp          time.Time  `json:"timestamp"`
	ChangeType         ChangeType `json:"change_type"`
	Description        string     `json:"description"`
	IdentityChanges    int        `json:"identity_changes"`
	UserProfileChanges int        `json:"user_profile_changes"`
}

// ExportVersion exports a version to a file
func (vm *VersionManager) ExportVersion(versionID string, exportPath string) error {
	version, err := vm.store.GetVersion(versionID)
	if err != nil {
		return err
	}

	_, err = json.MarshalIndent(version, "", "  ")
	if err != nil {
		return err
	}

	return nil // Write to file would go here
}

// ImportVersion imports a version from a file
func (vm *VersionManager) ImportVersion(data []byte) (*PersonaVersion, error) {
	var version PersonaVersion
	if err := json.Unmarshal(data, &version); err != nil {
		return nil, err
	}

	// Generate new ID to avoid conflicts
	version.ID = fmt.Sprintf("version_%d", time.Now().UnixNano())
	version.Timestamp = time.Now()

	if err := vm.store.SaveVersion(&version); err != nil {
		return nil, err
	}

	return &version, nil
}

// DisplayVersion returns a formatted string representation of a version
func (vm *VersionManager) DisplayVersion(version *PersonaVersion) string {
	var result string

	result += fmt.Sprintf("Version: %s\n", version.ID)
	result += strings.Repeat("=", len("Version: "+version.ID)) + "\n\n"

	result += fmt.Sprintf("Timestamp: %s\n", version.Timestamp.Format("2006-01-02 15:04:05"))
	result += fmt.Sprintf("Change Type: %s\n", version.ChangeType)
	result += fmt.Sprintf("Description: %s\n\n", version.ChangeDescription)

	if version.ProposalID != nil {
		result += fmt.Sprintf("From Proposal: %s\n\n", *version.ProposalID)
	}

	if version.Identity != nil {
		result += "Identity:\n"
		result += fmt.Sprintf("  Name: %s\n", version.Identity.Name)
		if version.Identity.Personality != "" {
			result += fmt.Sprintf("  Personality: %s\n", version.Identity.Personality)
		}
		if version.Identity.Voice != "" {
			result += fmt.Sprintf("  Voice: %s\n", version.Identity.Voice)
		}
		if len(version.Identity.Expertise) > 0 {
			result += fmt.Sprintf("  Expertise: %v\n", version.Identity.Expertise)
		}
		if len(version.Identity.Values) > 0 {
			result += fmt.Sprintf("  Values: %v\n", version.Identity.Values)
		}
		result += "\n"
	}

	if version.UserProfile != nil {
		result += "User Profile:\n"
		if version.UserProfile.Name != "" {
			result += fmt.Sprintf("  Name: %s\n", version.UserProfile.Name)
		}
		if version.UserProfile.CommunicationStyle != "" {
			result += fmt.Sprintf("  Communication Style: %s\n", version.UserProfile.CommunicationStyle)
		}
		if len(version.UserProfile.Expertise) > 0 {
			result += fmt.Sprintf("  Expertise: %v\n", version.UserProfile.Expertise)
		}
		if len(version.UserProfile.Goals) > 0 {
			result += fmt.Sprintf("  Goals: %v\n", version.UserProfile.Goals)
		}
		result += "\n"
	}

	return result
}

// DisplayVersionList returns a formatted list of versions
func (vm *VersionManager) DisplayVersionList(versions []*PersonaVersion) string {
	if len(versions) == 0 {
		return "No versions found.\n"
	}

	var result string
	result += fmt.Sprintf("Found %d version(s):\n\n", len(versions))

	for i, v := range versions {
		changeIcon := "✏️"
		switch v.ChangeType {
		case ChangeProposal:
			changeIcon = "💡"
		case ChangeRollback:
			changeIcon = "⏮️"
		case ChangeAuto:
			changeIcon = "🤖"
		}

		result += fmt.Sprintf("%d. %s [%s] %s\n",
			i+1, changeIcon, v.ChangeType, v.Timestamp.Format("2006-01-02 15:04"))
		result += fmt.Sprintf("   ID: %s\n", v.ID)
		result += fmt.Sprintf("   Description: %s\n", v.ChangeDescription)

		if i < len(versions)-1 {
			result += "\n"
		}
	}

	return result
}

// Ensure vm.store.db is accessible
func (vs *VersionStore) ensureDB() *gorm.DB {
	return vs.db
}

func (vm *VersionManager) GetStore() *VersionStore {
	return vm.store
}

func init() {
	// String multiplication helper
	_ = func(s string, count int) string {
		result := ""
		for i := 0; i < count; i++ {
			result += s
		}
		return result
	}
}
