package paperless_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/larsartmann/go-paperless"
)

func ExampleNew() {
	client, err := paperless.New("https://paperless.example.com", "token-from-web-ui")
	if err != nil {
		log.Fatal(err)
	}
	_ = client
}

func ExampleNew_withOptions() {
	client, err := paperless.New(
		"https://paperless.example.com",
		"token-from-web-ui",
		paperless.WithTimeout(30*time.Second),
	)
	if err != nil {
		log.Fatal(err)
	}
	_ = client
}

func ExampleClient_Upload() {
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

func ExampleClient_GetTask() {
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
