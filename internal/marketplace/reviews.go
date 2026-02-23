package marketplace

import (
	"context"
	"fmt"
	"time"

	"github.com/gmsas95/myrai-cli/internal/store"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ReviewsManager handles agent reviews and ratings
type ReviewsManager struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewReviewsManager creates a new reviews manager
func NewReviewsManager(db *gorm.DB, logger *zap.Logger) *ReviewsManager {
	return &ReviewsManager{
		db:     db,
		logger: logger,
	}
}

// SubmitReview submits a new review for an agent
func (rm *ReviewsManager) SubmitReview(ctx context.Context, agentID, userID string, rating int, title, content, version string) (*AgentReview, error) {
	// Validate rating
	if rating < 1 || rating > 5 {
		return nil, fmt.Errorf("rating must be between 1 and 5")
	}

	// Check if user has already reviewed this agent
	var existingReview AgentReview
	err := rm.db.Where("agent_id = ? AND user_id = ? AND is_deleted = ?",
		agentID, userID, false).First(&existingReview).Error

	if err == nil {
		// Update existing review
		existingReview.Rating = rating
		existingReview.Title = title
		existingReview.Content = content
		existingReview.Version = version
		existingReview.UpdatedAt = time.Now()

		if err := rm.db.Save(&existingReview).Error; err != nil {
			return nil, fmt.Errorf("failed to update review: %w", err)
		}

		rm.logger.Info("Updated review",
			zap.String("agent_id", agentID),
			zap.String("user_id", userID),
			zap.Int("rating", rating))

		// Recalculate rating
		rm.recalculateRating(agentID)

		return &existingReview, nil
	}

	// Create new review
	review := &AgentReview{
		AgentID:     agentID,
		UserID:      userID,
		Rating:      rating,
		Title:       title,
		Content:     content,
		Version:     version,
		IsPublished: true,
		IsDeleted:   false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := rm.db.Create(review).Error; err != nil {
		return nil, fmt.Errorf("failed to create review: %w", err)
	}

	rm.logger.Info("Submitted new review",
		zap.String("agent_id", agentID),
		zap.String("user_id", userID),
		zap.Int("rating", rating))

	// Recalculate rating
	rm.recalculateRating(agentID)

	return review, nil
}

// GetReviews retrieves reviews for an agent with pagination
func (rm *ReviewsManager) GetReviews(ctx context.Context, agentID string, limit, offset int) ([]*AgentReview, error) {
	var reviews []*AgentReview
	err := rm.db.Where("agent_id = ? AND is_published = ? AND is_deleted = ?",
		agentID, true, false).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&reviews).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get reviews: %w", err)
	}

	return reviews, nil
}

// GetUserReview gets a specific user's review for an agent
func (rm *ReviewsManager) GetUserReview(ctx context.Context, agentID, userID string) (*AgentReview, error) {
	var review AgentReview
	err := rm.db.Where("agent_id = ? AND user_id = ? AND is_deleted = ?",
		agentID, userID, false).First(&review).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get review: %w", err)
	}

	return &review, nil
}

// DeleteReview soft-deletes a review
func (rm *ReviewsManager) DeleteReview(ctx context.Context, reviewID, userID string) error {
	result := rm.db.Model(&AgentReview{}).
		Where("id = ? AND user_id = ?", reviewID, userID).
		Update("is_deleted", true)

	if result.Error != nil {
		return fmt.Errorf("failed to delete review: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("review not found or not owned by user")
	}

	// Get agent ID and recalculate
	var review AgentReview
	rm.db.Where("id = ?", reviewID).First(&review)
	rm.recalculateRating(review.AgentID)

	return nil
}

// MarkHelpful marks a review as helpful
func (rm *ReviewsManager) MarkHelpful(ctx context.Context, reviewID string) error {
	result := rm.db.Model(&AgentReview{}).
		Where("id = ?", reviewID).
		UpdateColumn("helpful", gorm.Expr("helpful + 1"))

	if result.Error != nil {
		return fmt.Errorf("failed to mark helpful: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("review not found")
	}

	return nil
}

// recalculateRating recalculates the average rating for an agent
func (rm *ReviewsManager) recalculateRating(agentID string) error {
	var result struct {
		AvgRating   float64
		ReviewCount int64
	}

	err := rm.db.Model(&AgentReview{}).
		Select("AVG(rating) as avg_rating, COUNT(*) as review_count").
		Where("agent_id = ? AND is_published = ? AND is_deleted = ?",
			agentID, true, false).
		Scan(&result).Error

	if err != nil {
		return err
	}

	// Update agent stats
	return rm.db.Model(&MarketplaceAgent{}).
		Where("id = ?", agentID).
		Updates(map[string]interface{}{
			"rating":       result.AvgRating,
			"review_count": result.ReviewCount,
		}).Error
}

// GetRatingSummary gets a rating summary for an agent
func (rm *ReviewsManager) GetRatingSummary(ctx context.Context, agentID string) (*RatingSummary, error) {
	summary := &RatingSummary{
		AgentID: agentID,
	}

	// Get average and count
	var result struct {
		AvgRating   float64
		ReviewCount int64
	}

	err := rm.db.Model(&AgentReview{}).
		Select("AVG(rating) as avg_rating, COUNT(*) as review_count").
		Where("agent_id = ? AND is_published = ? AND is_deleted = ?",
			agentID, true, false).
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	summary.AverageRating = result.AvgRating
	summary.TotalReviews = int(result.ReviewCount)

	// Get distribution
	var distribution []struct {
		Rating int
		Count  int64
	}

	err = rm.db.Model(&AgentReview{}).
		Select("rating, COUNT(*) as count").
		Where("agent_id = ? AND is_published = ? AND is_deleted = ?",
			agentID, true, false).
		Group("rating").
		Scan(&distribution).Error

	if err != nil {
		return nil, err
	}

	summary.Distribution = make(map[int]int)
	for _, d := range distribution {
		summary.Distribution[d.Rating] = int(d.Count)
	}

	return summary, nil
}

// RatingSummary provides aggregated rating information
type RatingSummary struct {
	AgentID       string      `json:"agent_id"`
	AverageRating float64     `json:"average_rating"`
	TotalReviews  int         `json:"total_reviews"`
	Distribution  map[int]int `json:"distribution"` // rating -> count
}

// GetRecentReviews gets recent reviews across all agents
func (rm *ReviewsManager) GetRecentReviews(ctx context.Context, limit int) ([]*AgentReview, error) {
	var reviews []*AgentReview
	err := rm.db.Where("is_published = ? AND is_deleted = ?", true, false).
		Order("created_at DESC").
		Limit(limit).
		Find(&reviews).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get recent reviews: %w", err)
	}

	return reviews, nil
}

// GetTopRatedAgents gets the top-rated agents
func (rm *ReviewsManager) GetTopRatedAgents(ctx context.Context, limit int) ([]*MarketplaceAgent, error) {
	var agents []*MarketplaceAgent
	err := rm.db.Where("review_count > 0").
		Order("rating DESC, review_count DESC").
		Limit(limit).
		Find(&agents).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get top rated agents: %w", err)
	}

	return agents, nil
}

// GetMostReviewedAgents gets the most reviewed agents
func (rm *ReviewsManager) GetMostReviewedAgents(ctx context.Context, limit int) ([]*MarketplaceAgent, error) {
	var agents []*MarketplaceAgent
	err := rm.db.Order("review_count DESC").
		Limit(limit).
		Find(&agents).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get most reviewed agents: %w", err)
	}

	return agents, nil
}

// ModerateReview allows admins to moderate reviews
func (rm *ReviewsManager) ModerateReview(ctx context.Context, reviewID string, publish bool) error {
	return rm.db.Model(&AgentReview{}).
		Where("id = ?", reviewID).
		Update("is_published", publish).Error
}

// EnsureReviewsManager creates the reviews manager from a store
func EnsureReviewsManager(s *store.Store, logger *zap.Logger) *ReviewsManager {
	return NewReviewsManager(s.DB(), logger)
}
