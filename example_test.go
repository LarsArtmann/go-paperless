package paperless_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/larsartmann/go-paperless"
)

func ExampleNew_invalidConfig() {
	_, err := paperless.New("", "token-from-web-ui") //art-dupl:accept doc-example boilerplate
	if errors.Is(err, paperless.ErrInvalidConfig) {
		fmt.Println("configure the base URL before retrying")
	}
	// Output: configure the base URL before retrying
}

func ExampleNew() {
	//art-dupl:accept doc-example boilerplate
	client, err := paperless.New("https://paperless.example.com", "token-from-web-ui")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(client != nil)
	// Output: true
}

func ExampleNew_withOptions() {
	client, err := paperless.New( //art-dupl:accept doc-example boilerplate
		"https://paperless.example.com",
		"token-from-web-ui",
		paperless.WithTimeout(30*time.Second),
		paperless.WithRetry(paperless.RetryPolicy{MaxAttempts: 4}),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(client != nil)
	// Output: true
}

//nolint:testableexamples // illustrative; running it would need a live server
func ExampleClient_WaitForTask() {
	//art-dupl:accept doc-example boilerplate
	client, err := paperless.New("https://paperless.example.com", "token-from-web-ui")
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Poll every 2 seconds (the DefaultTaskPollInterval) until the task
	// reaches a terminal state or the context expires.
	outcome, err := client.WaitForTask(
		ctx,
		"0198f7a2-9d3f-7c31-b5e4-5f2a9c1d8e77",
		paperless.DefaultTaskPollInterval,
	)
	if err != nil {
		log.Print(err)

		return
	}

	if documentID, inTrash, refused := outcome.Duplicate(); refused {
		fmt.Println("duplicate of", documentID, "in trash:", inTrash)

		return
	}

	fmt.Println("consumed as document", outcome.DocumentID)
}

//nolint:testableexamples // illustrative; running it would need a live server
func ExampleWithRetry() {
	client, err := paperless.New( //art-dupl:accept doc-example boilerplate
		"https://paperless.example.com",
		"token-from-web-ui",
		paperless.WithRetry(paperless.RetryPolicy{
			MaxAttempts:  5,
			InitialDelay: 200 * time.Millisecond,
			MaxDelay:     10 * time.Second,
			Multiplier:   2.0,
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	// Transient failures (network errors, 429/503, 5xx) retry with backoff;
	// a server Retry-After hint overrides the computed delay. Rejections
	// (401/403, other 4xx) fail fast without retrying.
	if err := client.Ping(ctx); err != nil {
		log.Fatal(err)
	}
}

//nolint:testableexamples // illustrative; running it would need a live server
func ExampleClient_Upload() {
	//art-dupl:accept doc-example boilerplate
	client, err := paperless.New("https://paperless.example.com", "token-from-web-ui")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	tagID, err := client.EnsureTag(ctx, "bank-sync")
	if err != nil {
		log.Fatal(err)
	}

	taskID, err := client.Upload(ctx, paperless.UploadRequest{
		Filename: "statement-2026-08.pdf",
		Content:  []byte("%PDF-1.4 ..."),
		Title:    "Statement August 2026",
		TagIDs:   []int{tagID},
	})
	if err != nil {
		log.Fatal(err)
	}

	outcome, found, err := client.GetTask(ctx, taskID)
	if err != nil {
		log.Fatal(err)
	}

	if found && outcome.Status.Terminal() {
		fmt.Println("consumed:", outcome.Status)
	}
}

//nolint:testableexamples // illustrative; running it would need a live server
func ExampleClient_GetTask() {
	//art-dupl:accept doc-example boilerplate
	client, err := paperless.New("https://paperless.example.com", "token-from-web-ui")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	outcome, found, err := client.GetTask(ctx, "0198f7a2-9d3f-7c31-b5e4-5f2a9c1d8e77")
	if err != nil {
		log.Fatal(err)
	}

	if found {
		fmt.Println(outcome.Status, outcome.DocumentID)
	}
}

// ExampleClient_EnsureTag mirrors the README quick start so signature
// drift between README and code breaks the build.
//
//nolint:testableexamples // illustrative; running it would need a live server
func ExampleClient_EnsureTag() {
	//art-dupl:accept doc-example boilerplate
	client, err := paperless.New("http://paperless.local:8000", "my-token")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	tagID, err := client.EnsureTag(ctx, "inboxclean")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(tagID)
}

// ExampleClient_ProbeCapabilities checks which checksum delivery a server
// uses before consuming its documents at scale.
//
//nolint:testableexamples // illustrative; running it would need a live server
func ExampleClient_ProbeCapabilities() {
	//art-dupl:accept doc-example boilerplate
	client, err := paperless.New("http://paperless.local:8000", "my-token")
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	caps, err := client.ProbeCapabilities(ctx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(caps.ChecksumShape(), caps.DocumentsSampled)
}
