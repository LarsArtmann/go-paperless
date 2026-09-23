package paperlesstest

import (
	"testing"

	paperless "github.com/larsartmann/go-paperless"
)

// newTestClient builds a real paperless.Client pointed at the fake server
// with the default test token. Round-tripping every fake route through the
// real client is the drift guard: the fake cannot silently diverge from
// what the SDK decodes.
func newTestClient(t testing.TB, server *Server) *paperless.Client {
	t.Helper()

	client, err := paperless.New(server.URL(), DefaultToken)
	if err != nil {
		t.Fatalf("paperless.New against fake: %v", err)
	}

	return client
}
