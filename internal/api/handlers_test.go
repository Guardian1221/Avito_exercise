package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Guardian1221/prsvc/internal/models"
	"github.com/Guardian1221/prsvc/internal/repo"
	"github.com/Guardian1221/prsvc/internal/service"
)

func setupTestService(t *testing.T) (*service.Service, func()) {
	r, err := repo.NewPostgresRepo("postgres://postgres:postgres@localhost:5432/prusr?sslmode=disable")
	if err != nil {
		t.Fatalf("failed to connect to db: %v", err)
	}

	svc := service.NewService(r)

	cleanup := func() {
		r.Close()
	}

	return svc, cleanup
}

func TestTeamEndpoints(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	handler := NewHandler(svc)

	team := models.Team{
		TeamName: "team1",
		Members: []models.TeamMember{
			{UserID: "user1", Username: "Alice", IsActive: true},
			{UserID: "user2", Username: "Bob", IsActive: true},
		},
	}
	body, _ := json.Marshal(team)
	req := httptest.NewRequest(http.MethodPost, "/team/add", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusCreated {
		t.Fatalf("/team/add failed: %s", w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/team/get?team_name=team1", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusOK {
		t.Fatalf("/team/get failed: %s", w.Body.String())
	}

	var gotTeam models.Team
	if err := json.NewDecoder(w.Body).Decode(&gotTeam); err != nil {
		t.Fatalf("failed to decode /team/get response: %v", err)
	}

	if gotTeam.TeamName != "team1" || len(gotTeam.Members) != 2 {
		t.Fatalf("/team/get returned wrong data: %+v", gotTeam)
	}
}

func TestPullRequestEndpoints(t *testing.T) {
	svc, cleanup := setupTestService(t)
	defer cleanup()

	handler := NewHandler(svc)

	author := models.TeamMember{UserID: "author1", Username: "Author", IsActive: true}
	team := models.Team{TeamName: "teamPR", Members: []models.TeamMember{author}}
	if err := svc.CreateTeam(context.Background(), team); err != nil {
		t.Fatalf("failed to create team: %v", err)
	}

	prReq := createPRReq{
		PullRequestID:   "pr1",
		PullRequestName: "Add feature X",
		AuthorID:        "author1",
	}
	body, _ := json.Marshal(prReq)
	req := httptest.NewRequest(http.MethodPost, "/pullRequest/create", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusCreated {
		t.Fatalf("/pullRequest/create failed: %s", w.Body.String())
	}

	var prResp map[string]models.PullRequest
	if err := json.NewDecoder(w.Body).Decode(&prResp); err != nil {
		t.Fatalf("failed to decode /pullRequest/create response: %v", err)
	}

	pr := prResp["pr"]
	if pr.PullRequestID != "pr1" {
		t.Fatalf("PR id mismatch: %+v", pr)
	}

	reassign := reassignReq{
		PullRequestID: "pr1",
		OldUserID:     author.UserID,
	}
	body, _ = json.Marshal(reassign)
	req = httptest.NewRequest(http.MethodPost, "/pullRequest/reassign", bytes.NewReader(body))
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Result().StatusCode != http.StatusConflict && w.Result().StatusCode != http.StatusOK {
		t.Fatalf("/pullRequest/reassign failed: %s", w.Body.String())
	}
}
