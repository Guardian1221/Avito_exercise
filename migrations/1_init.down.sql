DROP INDEX IF EXISTS idx_pr_reviewers_user;
DROP INDEX IF EXISTS idx_pr_status;
DROP INDEX IF EXISTS idx_pr_author;
DROP INDEX IF EXISTS idx_users_team_active;

DROP TABLE IF EXISTS pr_reviewers;
DROP TABLE IF EXISTS pull_requests;
DROP TYPE IF EXISTS pr_status;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS teams;
