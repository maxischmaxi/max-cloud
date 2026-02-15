package handler

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/max-cloud/api/internal/orchestrator"
	"github.com/max-cloud/api/internal/store"
	"github.com/max-cloud/shared/pkg/models"
)

// StreamLogs streamt Container-Logs als Server-Sent Events.
func (h *Handler) StreamLogs(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	svc, err := h.store.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			http.Error(w, `{"error":"service not found"}`, http.StatusNotFound)
			return
		}
		h.logger.Error("failed to get service for logs", "error", err, "id", id)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	follow, _ := strconv.ParseBool(r.URL.Query().Get("follow"))
	tail := int64(100)
	if v := r.URL.Query().Get("tail"); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil && parsed > 0 {
			tail = parsed
		}
	}

	rc, err := h.orchestrator.Logs(r.Context(), svc, orchestrator.LogsOptions{
		Follow: follow,
		Tail:   tail,
	})
	if err != nil {
		if errors.Is(err, orchestrator.ErrNoPods) {
			http.Error(w, `{"error":"no running pods found"}`, http.StatusServiceUnavailable)
			return
		}
		h.logger.Error("failed to get logs", "error", err, "id", id)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// WriteDeadline deaktivieren für langlebiges SSE-Streaming
	if ctrl := http.NewResponseController(w); ctrl != nil {
		ctrl.SetWriteDeadline(time.Time{})
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"streaming not supported"}`, http.StatusInternalServerError)
		return
	}

	scanner := bufio.NewScanner(rc)
	for scanner.Scan() {
		line := scanner.Text()
		stream := detectStream(line)
		entry := models.LogEntry{
			Timestamp: time.Now(),
			Message:   line,
			Stream:    stream,
		}
		data, err := json.Marshal(entry)
		if err != nil {
			continue
		}
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}

func detectStream(line string) string {
	var log map[string]interface{}
	if err := json.Unmarshal([]byte(line), &log); err != nil {
		return "stdout"
	}

	level := extractLevel(log)
	if level == "" {
		return "stdout"
	}

	switch level {
	case "error", "err", "fatal", "crit", "critical", "alert", "emerg", "emergency":
		return "stderr"
	default:
		return "stdout"
	}
}

func extractLevel(log map[string]interface{}) string {
	levelKeys := []string{"level", "severity", "lvl", "severity_text", "log_level"}
	for _, key := range levelKeys {
		if v, ok := log[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	return ""
}
