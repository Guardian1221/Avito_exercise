package models

import "time"

type Team struct {
	TeamName string       `db:"team_name" json:"team_name"`
	Members  []TeamMember `json:"members"`
}

type TeamMember struct {
	UserID   string `db:"user_id" json:"user_id"`
	Username string `db:"username" json:"username"`
	IsActive bool   `db:"is_active" json:"is_active"`
}

type User struct {
	UserID    string    `db:"user_id" json:"user_id"`
	Username  string    `db:"username" json:"username"`
	TeamName  string    `db:"team_name" json:"team_name"`
	IsActive  bool      `db:"is_active" json:"is_active"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type PullRequest struct {
	PullRequestID     string     `db:"pull_request_id" json:"pull_request_id"`
	PullRequestName   string     `db:"pull_request_name" json:"pull_request_name"`
	AuthorID          string     `db:"author_id" json:"author_id"`
	Status            string     `db:"status" json:"status"`
	AssignedReviewers []string   `json:"assigned_reviewers"`
	CreatedAt         time.Time  `db:"created_at" json:"createdAt,omitempty"`
	MergedAt          *time.Time `db:"merged_at" json:"mergedAt,omitempty"`
}
