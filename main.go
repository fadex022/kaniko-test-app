package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)

	addr := ":" + port
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, withAccessLog(mux)); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, "Hello from Kaniko-built container!")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Environment:")

	env := os.Environ()
	sort.Strings(env)
	for _, kv := range env {
		fmt.Fprintf(w, "  %s\n", kv)
	}
}

// withAccessLog wraps a handler and writes one log line per request once the
// response is complete. Includes method, path, status, byte count, duration,
// remote addr, and user-agent — enough to debug routing/auth issues from logs
// alone without needing tracing wired up.
func withAccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &recordingResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf(
			"%s %s %s %d %dB %s ua=%q",
			clientIP(r),
			r.Method,
			r.URL.RequestURI(),
			rec.status,
			rec.bytes,
			time.Since(start).Round(time.Microsecond),
			strings.ReplaceAll(r.UserAgent(), "\"", "'"),
		)
	})
}

type recordingResponseWriter struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

func (w *recordingResponseWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *recordingResponseWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		// http.ResponseWriter contract: a Write before WriteHeader implies 200.
		w.wroteHeader = true
	}
	n, err := w.ResponseWriter.Write(p)
	w.bytes += n
	return n, err
}

// clientIP prefers X-Forwarded-For (the gateway in front of this pod will set
// it) and falls back to the raw RemoteAddr. We split off the port for
// readability — the source port isn't useful in access logs.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}
	addr := r.RemoteAddr
	if i := strings.LastIndexByte(addr, ':'); i >= 0 {
		return addr[:i]
	}
	return addr
}
