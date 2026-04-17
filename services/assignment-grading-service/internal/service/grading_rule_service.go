package service

import (
	"context"
	"fmt"

	"slate/services/assignment-grading-service/internal/models"
	"slate/services/assignment-grading-service/internal/repository"
)

type gradingRuleService struct {
	repo repository.GradingRuleRepository
}

// NewGradingRuleService creates a new grading rule service
func NewGradingRuleService(repo repository.GradingRuleRepository) GradingRuleService {
	return &gradingRuleService{repo: repo}
}

// CreateRule creates a new grading rule
func (s *gradingRuleService) CreateRule(ctx context.Context, rule *models.GradingRule) (*models.GradingRule, error) {
	if err := rule.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if err := s.repo.Create(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to create grading rule: %w", err)
	}

	return rule, nil
}

// GetRule retrieves a grading rule by ID
func (s *gradingRuleService) GetRule(ctx context.Context, id string) (*models.GradingRule, error) {
	rule, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get grading rule: %w", err)
	}
	return rule, nil
}

// UpdateRule updates an existing grading rule
func (s *gradingRuleService) UpdateRule(ctx context.Context, rule *models.GradingRule) (*models.GradingRule, error) {
	if err := rule.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if err := s.repo.Update(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to update grading rule: %w", err)
	}

	return rule, nil
}

// DeleteRule deletes a grading rule
func (s *gradingRuleService) DeleteRule(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete grading rule: %w", err)
	}
	return nil
}

// ListRules lists grading rules for a tenant
func (s *gradingRuleService) ListRules(ctx context.Context, tenantID string) ([]*models.GradingRule, error) {
	rules, err := s.repo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list grading rules: %w", err)
	}
	return rules, nil
}
