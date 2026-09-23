package paperlesstest

import (
	"context"
	"errors"
	"testing"

	errorfamily "github.com/larsartmann/go-error-family"
	paperless "github.com/larsartmann/go-paperless"
)

func TestTokenEnforcementRejectsWrongTokens(t *testing.T) {
	t.Parallel()

	server := NewServer(t, WithToken("sekrit"))

	client, err := paperless.New(server.URL(), "wrong-token")
	if err != nil {
		t.Fatalf("paperless.New: %v", err)
	}

	pingErr := client.Ping(context.Background())
	if pingErr == nil {
		t.Fatal("Ping with a wrong token = nil error, want auth rejection")
	}

	if errorfamily.Classify(pingErr) != errorfamily.Rejection {
		t.Errorf("family = %v, want Rejection", errorfamily.Classify(pingErr))
	}

	coded, ok := errors.AsType[*errorfamily.Error](pingErr)
	if !ok || coded.Code() != "paperless.auth_failed" {
		t.Errorf("code = %v, want paperless.auth_failed", pingErr)
	}

	if got := len(server.Documents()); got != 0 {
		t.Errorf("documents = %d after unauthorized pings (endpoints must not run)", got)
	}
}

func TestTokenEnforcementAcceptsRightToken(t *testing.T) {
	t.Parallel()

	server := NewServer(t, WithToken("sekrit"))
	server.addDocument(Document{Title: "one", Checksum: "c1"})

	if err := newTestClientWithToken(t, server, "sekrit").Ping(context.Background()); err != nil {
		t.Errorf("Ping with the configured token: %v", err)
	}

	RequireAuthorized(t, server)
}

func TestRequireAuthorizedReportsWrongTokens(t *testing.T) {
	t.Parallel()

	stub := &tbStub{TB: t}

	server := NewServer(stub, WithToken("sekrit"))
	defer server.Close()

	client, err := paperless.New(server.URL(), "wrong-token")
	if err != nil {
		t.Fatalf("paperless.New: %v", err)
	}

	_ = client.Ping(context.Background())

	RequireAuthorized(stub, server)

	if stub.errorCount() == 0 {
		t.Error("RequireAuthorized = silent, want a report for the unauthorized request")
	}
}

func TestRequireUploadCount(t *testing.T) {
	t.Parallel()

	stub := &tbStub{TB: t}

	server := NewServer(stub)
	defer server.Close()

	client := newTestClientWithToken(t, server, DefaultToken)
	ctx := context.Background()

	_, err := client.Upload(
		ctx,
		paperless.UploadRequest{Filename: "a.pdf", Content: []byte("%PDF-a")},
	)
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	RequireUploadCount(stub, server, 1)

	if stub.errorCount() != 0 {
		t.Errorf("RequireUploadCount(1) reported: %v", stub.messages)
	}

	RequireUploadCount(stub, server, 2)

	if stub.errorCount() == 0 {
		t.Error("RequireUploadCount(2) = silent, want a report for 1 captured upload")
	}
}

func newTestClientWithToken(tb testing.TB, server *Server, token string) *paperless.Client {
	tb.Helper()

	client, err := paperless.New(server.URL(), token)
	if err != nil {
		tb.Fatalf("paperless.New against fake: %v", err)
	}

	return client
}
