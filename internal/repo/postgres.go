package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Guardian1221/prsvc/internal/models"
	"github.com/jmoiron/sqlx"

	_ "github.com/jackc/pgx/v4/stdlib"
)

type PostgresRepo struct {
	db *sqlx.DB
}

func NewPostgresRepo(dsn string) (*PostgresRepo, error) {
	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Minute * 30)
	return &PostgresRepo{db: db}, nil
}

func (r *PostgresRepo) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}


var ErrTeamExists = errors.New("team exists")

func (r *PostgresRepo) CreateTeam(ctx context.Context, t models.Team) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var exists bool
	err = tx.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM teams WHERE team_name=$1)", t.TeamName)
	if err != nil {
		tx.Rollback()
		return err
	}
	if exists {
		tx.Rollback()
		return ErrTeamExists
	}

	_, err = tx.ExecContext(ctx, "INSERT INTO teams(team_name) VALUES($1)", t.TeamName)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, m := range t.Members {
		_, err = tx.ExecContext(ctx, `
INSERT INTO users(user_id, username, team_name, is_active, created_at)
VALUES ($1,$2,$3,$4, now())
ON CONFLICT (user_id) DO UPDATE SET username = EXCLUDED.username, team_name = EXCLUDED.team_name, is_active = EXCLUDED.is_active
        `, m.UserID, m.Username, t.TeamName, m.IsActive)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *PostgresRepo) GetTeam(ctx context.Context, teamName string) (*models.Team, error) {
	var t models.Team
	t.TeamName = teamName
	var members []models.TeamMember
	if err := r.db.SelectContext(ctx, &members, "SELECT user_id, username, is_active FROM users WHERE team_name=$1 ORDER BY user_id", teamName); err != nil {
		if err == sql.ErrNoRows {
			members = []models.TeamMember{}
		} else {
			return nil, err
		}
	}
	var exists bool
	if err := r.db.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM teams WHERE team_name=$1)", teamName); err != nil {
		return nil, err
	}
	if !exists {
		return nil, sql.ErrNoRows
	}
	t.Members = members
	return &t, nil
}

func (r *PostgresRepo) SetUserIsActive(ctx context.Context, userID string, active bool) (*models.User, error) {
	res, err := r.db.ExecContext(ctx, "UPDATE users SET is_active=$1 WHERE user_id=$2", active, userID)
	if err != nil {
		return nil, err
	}
	cnt, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if cnt == 0 {
		return nil, sql.ErrNoRows
	}
	var u models.User
	if err := r.db.GetContext(ctx, &u, "SELECT user_id, username, team_name, is_active, created_at FROM users WHERE user_id=$1", userID); err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *PostgresRepo) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	var u models.User
	if err := r.db.GetContext(ctx, &u, "SELECT user_id, username, team_name, is_active, created_at FROM users WHERE user_id=$1", userID); err != nil {
		return nil, err
	}
	return &u, nil
}


var ErrPRExists = errors.New("pr exists")

func (r *PostgresRepo) CreatePullRequestWithReviewers(ctx context.Context, pr models.PullRequest, reviewers []string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var exists bool
	if err := tx.GetContext(ctx, &exists, "SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id=$1)", pr.PullRequestID); err != nil {
		tx.Rollback()
		return err
	}
	if exists {
		tx.Rollback()
		return ErrPRExists
	}

	_, err = tx.ExecContext(ctx, `
INSERT INTO pull_requests(pull_request_id, pull_request_name, author_id, status, created_at)
VALUES ($1,$2,$3,'OPEN', now())
`, pr.PullRequestID, pr.PullRequestName, pr.AuthorID)
	if err != nil {
		tx.Rollback()
		return err
	}

	for _, ruid := range reviewers {
		_, err = tx.ExecContext(ctx, "INSERT INTO pr_reviewers(pull_request_id, user_id) VALUES($1,$2)", pr.PullRequestID, ruid)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (r *PostgresRepo) GetPullRequest(ctx context.Context, prID string) (*models.PullRequest, error) {
	var pr models.PullRequest
	if err := r.db.GetContext(ctx, &pr, "SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at FROM pull_requests WHERE pull_request_id=$1", prID); err != nil {
		return nil, err
	}
	var revs []string
	if err := r.db.SelectContext(ctx, &revs, "SELECT user_id FROM pr_reviewers WHERE pull_request_id=$1 ORDER BY user_id", prID); err != nil {
		return nil, err
	}
	pr.AssignedReviewers = revs
	return &pr, nil
}

func (r *PostgresRepo) SelectRandomActiveTeamMembersExcluding(ctx context.Context, teamName string, exclude []string, limit int) ([]string, error) {
	var args []interface{}
	args = append(args, teamName)
	sb := strings.Builder{}
	sb.WriteString("SELECT user_id FROM users WHERE team_name=$1 AND is_active = true")
	if len(exclude) > 0 {
		sb.WriteString(" AND user_id NOT IN (")
		for i := range exclude {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString(fmt.Sprintf("$%d", i+2))
			args = append(args, exclude[i])
		}
		sb.WriteString(")")
	}
	sb.WriteString(" ORDER BY RANDOM() LIMIT ")
	sb.WriteString(fmt.Sprintf("%d", limit))

	var res []string
	if err := r.db.SelectContext(ctx, &res, sb.String(), args...); err != nil {
		return nil, err
	}
	return res, nil
}

var ErrPRMerged = errors.New("pr merged")
var ErrNotAssigned = errors.New("not assigned")
var ErrNoCandidate = errors.New("no candidate")

func (r *PostgresRepo) ReassignReviewer(ctx context.Context, prID string, oldReviewerID string) (string, *models.PullRequest, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", nil, err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	var prRow struct {
		PullRequestID string       `db:"pull_request_id"`
		AuthorID      string       `db:"author_id"`
		Status        string       `db:"status"`
		MergedAt      sql.NullTime `db:"merged_at"`
	}
	if err := tx.GetContext(ctx, &prRow, "SELECT pull_request_id, author_id, status, merged_at FROM pull_requests WHERE pull_request_id=$1 FOR UPDATE", prID); err != nil {
		if err == sql.ErrNoRows {
			tx.Rollback()
			return "", nil, sql.ErrNoRows
		}
		tx.Rollback()
		return "", nil, err
	}

	if prRow.Status == "MERGED" {
		tx.Rollback()
		return "", nil, ErrPRMerged
	}

	var count int
	if err := tx.GetContext(ctx, &count, "SELECT COUNT(1) FROM pr_reviewers WHERE pull_request_id=$1 AND user_id=$2", prID, oldReviewerID); err != nil {
		tx.Rollback()
		return "", nil, err
	}
	if count == 0 {
		tx.Rollback()
		return "", nil, ErrNotAssigned
	}

	var teamName string
	if err := tx.GetContext(ctx, &teamName, "SELECT team_name FROM users WHERE user_id=$1", oldReviewerID); err != nil {
		if err == sql.ErrNoRows {
			tx.Rollback()
			return "", nil, sql.ErrNoRows
		}
		tx.Rollback()
		return "", nil, err
	}

	var currentReviewers []string
	if err := tx.SelectContext(ctx, &currentReviewers, "SELECT user_id FROM pr_reviewers WHERE pull_request_id=$1", prID); err != nil {
		tx.Rollback()
		return "", nil, err
	}
	exclude := append(currentReviewers, prRow.AuthorID)

	args := []interface{}{teamName}
	sb := strings.Builder{}
	sb.WriteString("SELECT user_id FROM users WHERE team_name=$1 AND is_active = true")
	if len(exclude) > 0 {
		sb.WriteString(" AND user_id NOT IN (")
		for i := range exclude {
			if i > 0 {
				sb.WriteString(",")
			}
			sb.WriteString(fmt.Sprintf("$%d", i+2))
			args = append(args, exclude[i])
		}
		sb.WriteString(")")
	}
	sb.WriteString(" ORDER BY RANDOM() LIMIT 1")
	var candidate string
	row := tx.QueryRowxContext(ctx, sb.String(), args...)
	if err := row.Scan(&candidate); err != nil {
		if err == sql.ErrNoRows {
			tx.Rollback()
			return "", nil, ErrNoCandidate
		}
		tx.Rollback()
		return "", nil, err
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM pr_reviewers WHERE pull_request_id=$1 AND user_id=$2", prID, oldReviewerID); err != nil {
		tx.Rollback()
		return "", nil, err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO pr_reviewers(pull_request_id, user_id) VALUES($1,$2)", prID, candidate); err != nil {
		tx.Rollback()
		return "", nil, err
	}

	var updated models.PullRequest
	if err := tx.GetContext(ctx, &updated, "SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at FROM pull_requests WHERE pull_request_id=$1", prID); err != nil {
		tx.Rollback()
		return "", nil, err
	}
	var revs []string
	if err := tx.SelectContext(ctx, &revs, "SELECT user_id FROM pr_reviewers WHERE pull_request_id=$1 ORDER BY user_id", prID); err != nil {
		tx.Rollback()
		return "", nil, err
	}
	updated.AssignedReviewers = revs

	if err := tx.Commit(); err != nil {
		return "", nil, err
	}
	return candidate, &updated, nil
}

func (r *PostgresRepo) SelectInitialReviewers(ctx context.Context, authorID string, limit int) ([]string, error) {
	var team string
	if err := r.db.GetContext(ctx, &team, "SELECT team_name FROM users WHERE user_id=$1", authorID); err != nil {
		return nil, err
	}
	var res []string
	if err := r.db.SelectContext(ctx, &res, "SELECT user_id FROM users WHERE team_name=$1 AND is_active = true AND user_id <> $2 ORDER BY RANDOM() LIMIT $3", team, authorID, limit); err != nil {
		return nil, err
	}
	return res, nil
}
