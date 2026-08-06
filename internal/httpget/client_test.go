package httpget

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetReturnsBodyOn200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/quote/GOOGL:NASDAQ" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Write([]byte(`{"price":100}`))
	}))
	defer srv.Close()

	c := New(srv.URL)
	body, err := c.Get(context.Background(), "/v1/quote/GOOGL:NASDAQ")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if string(body) != `{"price":100}` {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestGetReturnsErrorOnNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := New(srv.URL)
	if _, err := c.Get(context.Background(), "/v1/quote/NOPE"); err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

func TestNewTrimsTrailingSlash(t *testing.T) {
	c := New("http://localhost:8080/")
	if c.BaseURL != "http://localhost:8080" {
		t.Fatalf("BaseURL not trimmed: %s", c.BaseURL)
	}
}

func TestGetSendsConfiguredHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got != "TestAgent/1.0" {
			t.Fatalf("User-Agent = %q", got)
		}
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := New(srv.URL)
	c.Header.Set("User-Agent", "TestAgent/1.0")
	if _, err := c.Get(context.Background(), "/x"); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
}

func TestDefaultTimeoutIs30Seconds(t *testing.T) {
	c := New("http://example.com")
	if c.Timeout != 30*time.Second {
		t.Fatalf("Timeout = %v, want 30s", c.Timeout)
	}
}

func TestGetReusesHTTPClient(t *testing.T) {
	c := New("http://example.com")
	h1 := c.httpClient()
	h2 := c.httpClient()
	if h1 != h2 {
		t.Fatalf("httpClient should be reused across Get calls")
	}
}

func TestConcurrentGetsAreSafe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := New(srv.URL)
	c.Header.Set("User-Agent", "TestAgent/1.0")

	const n = 16
	done := make(chan error, n)
	for i := 0; i < n; i++ {
		go func() {
			body, err := c.Get(context.Background(), "/")
			if err != nil {
				done <- err
				return
			}
			if string(body) != "ok" {
				done <- fmt.Errorf("body=%s", body)
				return
			}
			done <- nil
		}()
	}
	for i := 0; i < n; i++ {
		if err := <-done; err != nil {
			t.Fatalf("concurrent Get error: %v", err)
		}
	}
}
