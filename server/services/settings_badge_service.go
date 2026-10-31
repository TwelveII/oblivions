package services

import (
	"context"
	"fmt"

	"github.com/akinalp/mqvi/models"
	"github.com/akinalp/mqvi/repository"
)

// SettingsBadgeService drives the dot indicators in Settings for admin
// Feedback/Reports panels and the user's own Feedback panel.
type SettingsBadgeService interface {
	GetAdminBadges(ctx context.Context, user *models.User) (AdminBadgeState, error)
	GetUserFeedbackBadge(ctx context.Context, user *models.User) (bool, error)
	MarkFeedbackSeen(ctx context.Context, userID string) error
	MarkReportsSeen(ctx context.Context, userID string) error
}

type AdminBadgeState struct {
	HasNewFeedback bool `json:"has_new_feedback"`
	HasNewReports  bool `json:"has_new_reports"`
}

type settingsBadgeService struct {
	userRepo     repository.UserRepository
	feedbackRepo repository.FeedbackRepository
	reportRepo   repository.ReportRepository
}

func NewSettingsBadgeService(
	userRepo repository.UserRepository,
	feedbackRepo repository.FeedbackRepository,
	reportRepo repository.ReportRepository,
) SettingsBadgeService {
	return &settingsBadgeService{
		userRepo:     userRepo,
		feedbackRepo: feedbackRepo,
		reportRepo:   reportRepo,
	}
}

func (s *settingsBadgeService) GetAdminBadges(ctx context.Context, user *models.User) (AdminBadgeState, error) {
	if user == nil {
		return AdminBadgeState{}, nil
	}

	state := AdminBadgeState{}

	latestFB, err := s.feedbackRepo.LatestCreatedAt(ctx)
	if err != nil {
		return AdminBadgeState{}, fmt.Errorf("badges feedback: %w", err)
	}
	if latestFB != nil && (user.FeedbackLastSeenAt == nil || latestFB.After(*user.FeedbackLastSeenAt)) {
		state.HasNewFeedback = true
	}

	latestRep, err := s.reportRepo.LatestCreatedAt(ctx)
	if err != nil {
		return AdminBadgeState{}, fmt.Errorf("badges reports: %w", err)
	}
	if latestRep != nil && (user.ReportsLastSeenAt == nil || latestRep.After(*user.ReportsLastSeenAt)) {
		state.HasNewReports = true
	}

	return state, nil
}

func (s *settingsBadgeService) GetUserFeedbackBadge(ctx context.Context, user *models.User) (bool, error) {
	if user == nil {
		return false, nil
	}
	latest, err := s.feedbackRepo.LatestAdminReplyForUser(ctx, user.ID)
	if err != nil {
		return false, fmt.Errorf("user feedback badge: %w", err)
	}
	if latest == nil {
		return false, nil
	}
	return user.FeedbackLastSeenAt == nil || latest.After(*user.FeedbackLastSeenAt), nil
}

func (s *settingsBadgeService) MarkFeedbackSeen(ctx context.Context, userID string) error {
	return s.userRepo.MarkFeedbackSeen(ctx, userID)
}

func (s *settingsBadgeService) MarkReportsSeen(ctx context.Context, userID string) error {
	return s.userRepo.MarkReportsSeen(ctx, userID)
}
