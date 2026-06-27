package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/tianyl1984/spark/internal/runner"
)

const maxBodyBytes = 10 << 20 // 10 MiB

// payload captures the fields we need from a GitHub webhook body.
type payload struct {
	Repository struct {
		Name     string `json:"name"`
		FullName string `json:"full_name"`
	} `json:"repository"`
}

// Handler verifies GitHub webhooks and dispatches project scripts.
type Handler struct {
	Secret string
	Runner *runner.Runner
}

// New creates a webhook Handler.
func New(secret string, r *runner.Runner) *Handler {
	return &Handler{Secret: secret, Runner: r}
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
	if err != nil {
		http.Error(w, "read body", http.StatusBadRequest)
		return
	}

	if !h.verifySignature(r, body) {
		log.Printf("rejected request: invalid signature")
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	event := r.Header.Get("X-GitHub-Event")
	if event == "ping" {
		writeJSON(w, http.StatusOK, map[string]string{"msg": "pong"})
		return
	}

	var p payload
	if err := json.Unmarshal(body, &p); err != nil {
		http.Error(w, "parse payload", http.StatusBadRequest)
		return
	}

	project := p.Repository.Name
	if project == "" {
		http.Error(w, "no repository in payload", http.StatusBadRequest)
		return
	}

	if !h.Runner.HasProject(project) {
		log.Printf("no config for project %q (event=%s)", project, event)
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored", "project": project})
		return
	}

	log.Printf("dispatching project %q (event=%s)", project, event)
	// Run asynchronously so GitHub gets a fast response.
	go func() {
		if err := h.Runner.Run(context.Background(), project); err != nil {
			log.Printf("[%s] run finished with error: %v", project, err)
		}
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted", "project": project})
}

// verifySignature checks the X-Hub-Signature-256 header against the secret.
// When no secret is configured, verification is skipped.
func (h *Handler) verifySignature(r *http.Request, body []byte) bool {
	if h.Secret == "" {
		return true
	}
	sig := r.Header.Get("X-Hub-Signature-256")
	if !strings.HasPrefix(sig, "sha256=") {
		return false
	}
	want := strings.TrimPrefix(sig, "sha256=")

	mac := hmac.New(sha256.New, []byte(h.Secret))
	mac.Write(body)
	got := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(want), []byte(got))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
