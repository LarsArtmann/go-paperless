package paperless

import (
	"encoding/json/v2"
	"strings"
	"testing"
	"time"
)

func FuzzParseRetryAfter(f *testing.F) {
	f.Add("", int64(0))
	f.Add("7", int64(0))
	f.Add("-3", int64(0))
	f.Add("0", int64(0))
	f.Add("2026-09-13T12:00:00Z", int64(1789000000))
	f.Add("Sun, 13 Sep 2026 12:00:00 GMT", int64(1789000000))
	f.Add("not-a-date", int64(1789000000))
	f.Add(" 42 ", int64(1789000000))

	f.Fuzz(func(t *testing.T, value string, nowUnix int64) {
		now := time.Unix(nowUnix, 0)

		delay, ok := parseRetryAfter(value, now)
		if !ok {
			return
		}

		if delay < 0 {
			t.Fatalf("parseRetryAfter(%q) accepted a negative delay %s", value, delay)
		}
	})
}

func FuzzParseDocumentCreated(f *testing.F) {
	f.Add("")
	f.Add("2026-08-18")
	f.Add("2026-08-18T10:00:00Z")
	f.Add("2026-08-18T10:00:00+02:00")
	f.Add("2026-08-18T10:00:00")
	f.Add("not a date")
	f.Add("2026-13-45")

	f.Fuzz(func(t *testing.T, raw string) {
		parsed := parseDocumentCreated(raw)
		if parsed.IsZero() {
			return
		}

		if reparsed := parseDocumentCreated(parsed.Format(time.RFC3339)); reparsed != parsed {
			t.Fatalf("parseDocumentCreated is not stable for %q: %v then %v", raw, parsed, reparsed)
		}
	})
}

func FuzzChecksumFrom(f *testing.F) {
	f.Add("", []byte(nil), []byte(""))
	f.Add("flat-checksum", []byte{1}, []byte("ignored-when-flat"))
	f.Add("", []byte{1}, []byte("root-checksum"))
	f.Add("", []byte{0, 1}, []byte("child\x00root"))
	f.Add("", []byte{1, 1}, []byte("a\x00b"))
	f.Add("", []byte{0}, []byte(""))

	f.Fuzz(func(t *testing.T, flat string, rootFlags []byte, checksumBlob []byte) {
		rawChecksums := strings.Split(string(checksumBlob), "\x00")
		versions := make([]documentVersionPayload, 0, len(rootFlags))
		for i := 0; i < len(rootFlags) && i < len(rawChecksums); i++ {
			versions = append(versions, documentVersionPayload{
				Checksum: rawChecksums[i],
				IsRoot:   rootFlags[i] != 0,
			})
		}

		got := checksumFrom(flat, versions)

		if flat != "" {
			if got != flat {
				t.Fatalf("checksumFrom(%q, ...) = %q, want the flat checksum verbatim", flat, got)
			}

			return
		}

		if got == "" {
			return
		}

		for _, checksum := range rawChecksums {
			if checksum == got {
				return
			}
		}

		t.Fatalf("checksumFrom(\"\", ...) = %q, which no version carries (roots=%v checksums=%q)",
			got, rootFlags, rawChecksums)
	})
}

func FuzzClassifyTask(f *testing.F) {
	f.Add([]byte(`{"task_id":"t1","status":"success","result_data":{"document_id":42}}`))
	f.Add([]byte(`{"task_id":"t2","status":"failure","result_data":{"duplicate_of":7,"duplicate_in_trash":true}}`))
	f.Add([]byte(`{"task_id":"t3","status":"failure","result_data":{"error_message":"bad pdf"}}`))
	f.Add([]byte(`{"task_id":"t4","status":"PENDING","related_document_ids":[9,8]}`))
	f.Add([]byte(`{`))
	f.Add([]byte(`{"result_data":{"document_id":null,"duplicate_of":null}}`))

	f.Fuzz(func(t *testing.T, payloadJSON []byte) {
		var payload taskPayload
		if err := json.Unmarshal(payloadJSON, &payload); err != nil {
			t.Skip()
		}

		outcome := classifyTask(payload)

		switch {
		case payload.ResultData.DocumentID != nil:
			if outcome.DuplicateRefused || outcome.DocumentID != *payload.ResultData.DocumentID {
				t.Fatalf("document_id %d misclassified: %+v", *payload.ResultData.DocumentID, outcome)
			}
		case payload.ResultData.DuplicateOf != nil:
			if !outcome.DuplicateRefused ||
				outcome.DocumentID != *payload.ResultData.DuplicateOf ||
				outcome.DuplicateInTrash != payload.ResultData.DuplicateInTrash {
				t.Fatalf("duplicate_of %d misclassified: %+v", *payload.ResultData.DuplicateOf, outcome)
			}
		case len(payload.RelatedDocumentIDs) > 0:
			if outcome.DocumentID != payload.RelatedDocumentIDs[0] {
				t.Fatalf("related_document_ids[0] %d misclassified: %+v",
					payload.RelatedDocumentIDs[0], outcome)
			}
		}

		wantMessage := ""
		if payload.ResultData.ErrorMessage != nil {
			wantMessage = *payload.ResultData.ErrorMessage
		}

		if outcome.ErrorMessage != wantMessage {
			t.Fatalf("ErrorMessage = %q, want %q", outcome.ErrorMessage, wantMessage)
		}
	})
}
