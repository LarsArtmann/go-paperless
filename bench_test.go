package paperless

import (
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"testing"
)

// BenchmarkEnsureTag measures the existing-tag path (the find-then-reuse
// walk: one paginated tag scan per call) — papersync's per-upload ensure
// makes this the hot lookup against small self-hosted servers.
func BenchmarkEnsureTag(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; version=10")

		_, _ = w.Write(
			[]byte(
				`{"results": [{"id": 7, "name": "inboxclean", "matching_algorithm": 0, "slug": "inboxclean"}], "count": 1, "next": null, "previous": null}`,
			),
		)
	}))
	defer server.Close()

	client, err := New(server.URL, "token")
	if err != nil {
		b.Fatalf("New: %v", err)
	}

	ctx := b.Context()

	b.ResetTimer()

	for b.Loop() {
		if _, err := client.EnsureTag(ctx, "inboxclean"); err != nil {
			b.Fatalf("EnsureTag: %v", err)
		}
	}
}

// BenchmarkBrandedIDMarshal measures the branded-ID hot path consumers
// pay on every decode: a DocumentID field inside a JSON round trip (the
// ADR 0004 wrapper over the raw value).
func BenchmarkBrandedIDMarshal(b *testing.B) {
	type envelope struct {
		ID DocumentID `json:"id"`
	}

	out, err := json.Marshal(envelope{ID: NewDocumentID(42)})
	if err != nil {
		b.Fatalf("marshal: %v", err)
	}

	b.ResetTimer()

	for b.Loop() {
		var decoded envelope

		if err := json.Unmarshal(out, &decoded); err != nil {
			b.Fatalf("unmarshal: %v", err)
		}

		if decoded.ID.Get() != 42 {
			b.Fatalf("round trip lost the value: %d", decoded.ID.Get())
		}
	}
}

// BenchmarkBrandedIDConstructGet measures the pure wrapper cost (no
// JSON): construction from a raw value and reading it back — the shape
// consumer loops run per row.
func BenchmarkBrandedIDConstructGet(b *testing.B) {
	var sink int64

	for b.Loop() {
		sink = NewDocumentID(int64(b.N) & 0x7fffffff).Get()
	}

	if sink < 0 {
		b.Fatal("unreachable: keeps the loop from optimizing away")
	}
}

// Periodic benchmark recipe (no code): run weekly or before every
// release
//
//	go test -bench 'BenchmarkListChecksums|BenchmarkEnsureTag|BenchmarkBrandedID' -benchmem -count=10 . | tee bench.txt
//
// and compare against the previous recording with benchstat. Baselines
// live in the release PR description, never in the repo — numbers drift
// across machines.
