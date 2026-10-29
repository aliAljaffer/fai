package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

const eventHeartbeat = 25 * time.Second

// eventHub fans mutation notifications out to every open SSE stream. Buffered
// channels let a slow reader lag briefly; further events are dropped for it
// instead of blocking the writer.
type eventHub struct {
	mu          sync.Mutex
	subscribers map[chan []byte]struct{}
}

func newEventHub() *eventHub {
	return &eventHub{subscribers: make(map[chan []byte]struct{})}
}

func (h *eventHub) subscribe() chan []byte {
	ch := make(chan []byte, 16)
	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *eventHub) unsubscribe(ch chan []byte) {
	h.mu.Lock()
	if _, ok := h.subscribers[ch]; ok {
		delete(h.subscribers, ch)
		close(ch)
	}
	h.mu.Unlock()
}

func (h *eventHub) publish(payload []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subscribers {
		select {
		case ch <- payload:
		default:
		}
	}
}

func (s *Server) hub() *eventHub {
	s.hubOnce.Do(func() {
		s.hubValue = newEventHub()
	})
	return s.hubValue
}

func (s *Server) broadcast(topic string) {
	payload, err := json.Marshal(map[string]string{"topic": topic})
	if err != nil {
		return
	}
	s.hub().publish(payload)
}

func eventTopic(path string) string {
	topic := strings.TrimPrefix(path, "/api/")
	topic = strings.TrimPrefix(topic, "/")
	if i := strings.IndexByte(topic, '/'); i >= 0 {
		topic = topic[:i]
	}
	return topic
}

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	ch := s.hub().subscribe()
	defer s.hub().unsubscribe(ch)
	heartbeat := time.NewTicker(eventHeartbeat)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case payload, open := <-ch:
			if !open {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", payload)
			flusher.Flush()
		case <-heartbeat.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusRecorder) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (s *Server) broadcastMutations(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		if recorder.status >= 200 && recorder.status < 300 {
			s.broadcast(eventTopic(r.URL.Path))
		}
	})
}
