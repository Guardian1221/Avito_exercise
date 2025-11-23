package service

import (
	"context"
	"database/sql"

	"github.com/Guardian1221/prsvc/internal/models"
	"github.com/Guardian1221/prsvc/internal/repo"
)

type Service struct {
	repo *repo.PostgresRepo
}

func (s *Service) GetTeam(ctx context.Context, name string) (*models.Team, error) {
	return s.repo.GetTeam(ctx, name)
}

func NewService(r *repo.PostgresRepo) *Service {
	return &Service{repo: r}
}

var (
	ErrTeamExists  = repo.ErrTeamExists
	ErrPRExists    = repo.ErrPRExists
	ErrPRMerged    = repo.ErrPRMerged
	ErrNotAssigned = repo.ErrNotAssigned
	ErrNoCandidate = repo.ErrNoCandidate
)

func (s *Service) CreateTeam(ctx context.Context, t models.Team) error {
	return s.repo.CreateTeam(ctx, t)
}

func (s *Service) CreatePullRequest(ctx context.Context, pr models.PullRequest) (*models.PullRequest, error) {
	_, err := s.repo.GetUserByID(ctx, pr.AuthorID)
	if err != nil {
		return nil, err
	}

	reviewers, err := s.repo.SelectInitialReviewers(ctx, pr.AuthorID, 2)
	if err != nil {
		return nil, err
	}

	if err := s.repo.CreatePullRequestWithReviewers(ctx, pr, reviewers); err != nil {
		return nil, err
	}

	created, err := s.repo.GetPullRequest(ctx, pr.PullRequestID)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Service) ReassignReviewer(ctx context.Context, prID string, oldReviewer string) (string, *models.PullRequest, error) {
	newID, pr, err := s.repo.ReassignReviewer(ctx, prID, oldReviewer)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil, err
		}
		return "", nil, err
	}
	return newID, pr, nil
}
