package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPFetch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"prefixes": [
				{"ipv4Prefix": "192.168.0.0/16"},
				{"ipv6Prefix": "2001:db8::/32"}
			]
		}`))
	}))
	defer ts.Close()

	s := NewHTTP("test", "good", ts.URL)
	prefixes, meta, err := s.Fetch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if meta.Source != "test" {
		t.Errorf("expected test, got %s", meta.Source)
	}

	if len(prefixes) != 2 {
		t.Errorf("expected 2 prefixes, got %d", len(prefixes))
	}
}
