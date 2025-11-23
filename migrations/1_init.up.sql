CREATE TABLE teams (
  team_name TEXT PRIMARY KEY
);

CREATE TABLE users (
  user_id TEXT PRIMARY KEY,
  username TEXT NOT NULL,
  team_name TEXT NOT NULL REFERENCES teams(team_name) ON DELETE CASCADE,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TYPE pr_status AS ENUM ('OPEN','MERGED');

CREATE TABLE pull_requests (
  pull_request_id TEXT PRIMARY KEY,
  pull_request_name TEXT NOT NULL,
  author_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE RESTRICT,
  status pr_status NOT NULL DEFAULT 'OPEN',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  merged_at TIMESTAMPTZ NULL
);

CREATE TABLE pr_reviewers (
  pull_request_id TEXT NOT NULL REFERENCES pull_requests(pull_request_id) ON DELETE CASCADE,
  user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE RESTRICT,
  PRIMARY KEY (pull_request_id, user_id)
);

CREATE INDEX idx_users_team_active ON users(team_name, is_active);
CREATE INDEX idx_pr_author ON pull_requests(author_id);
CREATE INDEX idx_pr_status ON pull_requests(status);
CREATE INDEX idx_pr_reviewers_user ON pr_reviewers(user_id);
