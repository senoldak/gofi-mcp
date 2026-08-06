package goficlient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
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
	got, err := c.Get(context.Background(), "/v1/quote/GOOGL:NASDAQ")
	if got == nil && err == nil {
		t.Fatal("expected a request to succeed against a trimmable base")
	}
}
