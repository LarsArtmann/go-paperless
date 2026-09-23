package paperlesstest

import (
	"context"
	"net/http"
	"testing"

	paperless "github.com/larsartmann/go-paperless"
)

// newTestClient builds a real paperless.Client pointed at the fake server
// with the default test token. Round-tripping every fake route through the
// real client is the drift guard: the fake cannot silently diverge from
// what the SDK decodes.
func newTestClient(tb testing.TB, server *Server) *paperless.Client {
	tb.Helper()

	client, err := paperless.New(server.URL(), DefaultToken)
	if err != nil {
		tb.Fatalf("paperless.New against fake: %v", err)
	}

	return client
}

// newTestRequest builds a context-backed bodyless request against the fake,
// keeping tests noctx-clean.
func newTestRequest(tb testing.TB, method, url string) *http.Request {
	tb.Helper()

	request, err := http.NewRequestWithContext(context.Background(), method, url, http.NoBody)
	if err != nil {
		tb.Fatalf("build %s %s: %v", method, url, err)
	}

	return request
}
