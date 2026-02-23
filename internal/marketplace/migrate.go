package marketplace

import (
	"fmt"

	"gorm.io/gorm"
)

// Migrate creates the marketplace tables in the database
func Migrate(db *gorm.DB) error {
	// Auto-migrate all marketplace models
	if err := db.AutoMigrate(
		&MarketplaceAgent{},
		&InstalledAgent{},
		&AgentReview{},
		&AgentVersion{},
		&AgentCategory{},
	); err != nil {
		return fmt.Errorf("failed to migrate marketplace tables: %w", err)
	}

	return nil
}

// MigrateWithStore creates the marketplace tables using a store wrapper
func MigrateWithStore(store interface{ DB() *gorm.DB }) error {
	return Migrate(store.DB())
}
