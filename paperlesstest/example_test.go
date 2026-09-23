package paperlesstest

import (
	"context"
	"fmt"

	paperless "github.com/larsartmann/go-paperless"
)

// Example_uploadLoop runs the full loop a consumer pipeline performs
// against the fake: upload, wait for the consumption task, verify the
// document landed in the list.
func Example_uploadLoop() {
	server := NewServer(&tbStub{})

	client, err := paperless.New(server.URL(), DefaultToken)
	if err != nil {
		fmt.Println("client:", err)

		return
	}

	ctx := context.Background()

	taskID, err := client.Upload(ctx, paperless.UploadRequest{
		Filename: "invoice.pdf",
		Content:  []byte("%PDF-invoice"),
	})
	if err != nil {
		fmt.Println("upload:", err)

		return
	}

	outcome, err := client.WaitForTask(ctx, taskID, 1)
	if err != nil {
		fmt.Println("wait:", err)

		return
	}

	checksums, err := client.ListDocumentChecksums(ctx)
	if err != nil {
		fmt.Println("list:", err)

		return
	}

	_, present := checksums[ChecksumOf([]byte("%PDF-invoice"))]
	fmt.Println(outcome.Status, present)

	server.Close()

	// Output: success true
}
