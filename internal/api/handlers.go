package api

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/Guardian1221/prsvc/internal/models"
	"github.com/Guardian1221/prsvc/internal/repo"
	"github.com/Guardian1221/prsvc/internal/service"
)

type Handler struct {
	svc *service.Service
}

func NewHandler(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/team/add":
		h.handleTeamAdd(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/team/get":
		h.handleTeamGet(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/pullRequest/create":
		h.handlePRCreate(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/pullRequest/reassign":
		h.handlePRReassign(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/health":
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	case r.Method == http.MethodGet && r.URL.Path == "/":
		w.WriteHeader(http.StatusOK)
	default:
		http.NotFound(w, r)
	}
}

func withTimeoutContext(r *http.Request) (context.Context, context.CancelFunc) {
	timeout := 5 * time.Second
	ctx := r.Context()
	return context.WithTimeout(ctx, timeout)
}

func (h *Handler) handleTeamAdd(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := withTimeoutContext(r)
	defer cancel()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "NOT_FOUND", "invalid body")
		return
	}
	var t models.Team
	if err := json.Unmarshal(body, &t); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "NOT_FOUND", "invalid json")
		return
	}

	if t.TeamName == "" {
		writeErrorJSON(w, http.StatusBadRequest, "NOT_FOUND", "team_name required")
		return
	}

	if err := h.svc.CreateTeam(ctx, t); err != nil {
		if err == repo.ErrTeamExists {
			writeErrorJSON(w, http.StatusBadRequest, "TEAM_EXISTS", "team_name already exists")
			return
		}
		log.Printf("CreateTeam error: %v", err)
		writeErrorJSON(w, http.StatusInternalServerError, "NOT_FOUND", "internal error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"team": t})
}

func (h *Handler) handleTeamGet(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := withTimeoutContext(r)
	defer cancel()

	q := r.URL.Query().Get("team_name")
	if q == "" {
		writeErrorJSON(w, http.StatusBadRequest, "NOT_FOUND", "team_name required")
		return
	}
	t, err := h.svc.GetTeam(ctx, q)
	if err != nil {
		writeErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "team not found")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

type createPRReq struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
}

func (h *Handler) handlePRCreate(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := withTimeoutContext(r)
	defer cancel()

	var req createPRReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "NOT_FOUND", "invalid json")
		return
	}
	if req.PullRequestID == "" || req.PullRequestName == "" || req.AuthorID == "" {
		writeErrorJSON(w, http.StatusBadRequest, "NOT_FOUND", "missing fields")
		return
	}

	pr := models.PullRequest{
		PullRequestID:   req.PullRequestID,
		PullRequestName: req.PullRequestName,
		AuthorID:        req.AuthorID,
	}
	created, err := h.svc.CreatePullRequest(ctx, pr)
	if err != nil {
		if err == repo.ErrPRExists {
			writeErrorJSON(w, http.StatusConflict, "PR_EXISTS", "PR id already exists")
			return
		}
		if err == nil {
		}
		writeErrorJSON(w, http.StatusInternalServerError, "NOT_FOUND", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"pr": created})
}

type reassignReq struct {
	PullRequestID string `json:"pull_request_id"`
	OldUserID     string `json:"old_user_id"`
}

func (h *Handler) handlePRReassign(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := withTimeoutContext(r)
	defer cancel()

	var req reassignReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "NOT_FOUND", "invalid json")
		return
	}
	if req.PullRequestID == "" || req.OldUserID == "" {
		writeErrorJSON(w, http.StatusBadRequest, "NOT_FOUND", "missing fields")
		return
	}

	newID, pr, err := h.svc.ReassignReviewer(ctx, req.PullRequestID, req.OldUserID)
	if err != nil {
		switch err {
		case repo.ErrPRMerged:
			writeErrorJSON(w, http.StatusConflict, "PR_MERGED", "cannot reassign on merged PR")
			return
		case repo.ErrNotAssigned:
			writeErrorJSON(w, http.StatusConflict, "NOT_ASSIGNED", "reviewer is not assigned to this PR")
			return
		case repo.ErrNoCandidate:
			writeErrorJSON(w, http.StatusConflict, "NO_CANDIDATE", "no active replacement candidate in team")
			return
		default:
			if err.Error() == "sql: no rows in result set" || err.Error() == "no rows in result set" {
				writeErrorJSON(w, http.StatusNotFound, "NOT_FOUND", "pr or user not found")
				return
			}
			log.Printf("reassign error: %v", err)
			writeErrorJSON(w, http.StatusInternalServerError, "NOT_FOUND", "internal error")
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"pr": pr, "replaced_by": newID})
}
