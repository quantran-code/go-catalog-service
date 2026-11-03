package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestRequestsAreLoggedWithTheRequestID(t *testing.T) {
	t.Run("carrying a request id", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		t.Cleanup(func() { log.SetOutput(os.Stderr) })

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set(RequestIDHeader, "abc")
		NewRouter().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status %d", rec.Code)
		}
		if got := rec.Header().Get(RequestIDHeader); got != "abc" {
			t.Fatalf("response header X-Request-ID = %q, want %q", got, "abc")
		}
		logs := buf.String()
		if got := strings.Count(logs, "request_id=abc"); got != 1 {
			t.Fatalf("found %d lines with request_id=abc in logs: %q", got, logs)
		}
		if !strings.Contains(logs, "method=GET") {
			t.Fatalf("log line is missing the method: %q", logs)
		}
		if !strings.Contains(logs, "path=/health") {
			t.Fatalf("log line is missing the path: %q", logs)
		}
	})

	t.Run("without a request id", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)
		t.Cleanup(func() { log.SetOutput(os.Stderr) })

		rec := httptest.NewRecorder()
		NewRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("status %d", rec.Code)
		}
		if got := rec.Header().Get(RequestIDHeader); got != "" {
			t.Fatalf("response header X-Request-ID = %q, want it to be absent", got)
		}
		logs := buf.String()
		if !strings.Contains(logs, "request_id=") || !strings.Contains(logs, "path=/health") {
			t.Fatalf("no log line written for an unidentified request: %q", logs)
		}
	})
}
