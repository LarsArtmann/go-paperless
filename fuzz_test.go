package paperless

import (
	"encoding/json/v2"
	"slices"
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

		if reparsed := parseDocumentCreated(parsed.Format(time.RFC3339)); !reparsed.Equal(parsed) {
			t.Fatalf("parseDocumentCreated is not stable for %q: %v then %v", raw, parsed, reparsed)
		}
	})
}

func FuzzDecodeSavedViewPayload(f *testing.F) {
	f.Add([]byte(`{"id":5,"name":"Inbox","show_on_dashboard":true,"show_in_sidebar":false,` +
		`"sort_field":"created","sort_reverse":true,` +
		`"filter_rules":[{"rule_type":6,"value":"has_tag:1"}]}`))
	f.Add([]byte(`{"id":1,"name":"","filter_rules":[]}`))
	f.Add([]byte(`{"id":-3,"name":"x","filter_rules":[{"rule_type":-1,"value":""}]}`))
	f.Add([]byte(`{"id":2,"name":"x","unknown_field":true,"filter_rules":null}`))
	f.Add([]byte(`{"filter_rules":[{"rule_type":1},{"rule_type":2,"value":"v"}]}`))
	f.Add([]byte(`{`))
	f.Add([]byte(`[]`))

	f.Fuzz(func(t *testing.T, payloadJSON []byte) {
		var payload savedViewPayload
		if err := json.Unmarshal(payloadJSON, &payload); err != nil {
			t.Skip()
		}

		view := payload.savedView()

		if view.ID != payload.ID || view.Name != payload.Name {
			t.Fatalf("core fields lost: payload %+v vs view %+v", payload, view)
		}

		if view.ShowOnDashboard != payload.ShowOnDashboard ||
			view.ShowInSidebar != payload.ShowInSidebar ||
			view.SortField != payload.SortField || view.SortReverse != payload.SortReverse {
			t.Fatalf("display fields lost: payload %+v vs view %+v", payload, view)
		}

		if len(view.FilterRules) != len(payload.FilterRules) {
			t.Fatalf("filter rules = %d, want %d", len(view.FilterRules), len(payload.FilterRules))
		}

		for i, rule := range view.FilterRules {
			source := payload.FilterRules[i]
			if rule.RuleType != SavedViewRuleType(source.RuleType) || rule.Value != source.Value {
				t.Fatalf("rule %d = %+v, want type %d value %q",
					i, rule, source.RuleType, source.Value)
			}
		}
	})
}

func FuzzDecodeShareLinkPayload(f *testing.F) {
	f.Add([]byte(`{"id":9,"created":"2026-09-14T10:00:00Z","expiration":"2027-01-01T00:00:00Z",` +
		`"slug":"abc123","document":42,"file_version":"archive"}`))
	f.Add([]byte(`{"id":1,"created":"2026-09-14T10:00:00Z","slug":"s","document":1,` +
		`"file_version":"original"}`))
	f.Add([]byte(`{"id":2,"slug":"","document":0,"file_version":""}`))
	f.Add([]byte(`{"id":3,"slug":"s","document":5,"file_version":"weird-version"}`))
	f.Add([]byte(`{"slug":"s"}`))
	f.Add([]byte(`{`))
	f.Add([]byte(`null`))

	f.Fuzz(func(t *testing.T, payloadJSON []byte) {
		var payload shareLinkPayload
		if err := json.Unmarshal(payloadJSON, &payload); err != nil {
			t.Skip()
		}

		link := payload.shareLink()

		if link.ID != payload.ID || link.Slug != payload.Slug ||
			link.DocumentID != payload.Document {
			t.Fatalf("core fields lost: payload %+v vs link %+v", payload, link)
		}

		if link.Created != payload.Created {
			t.Fatalf("created lost: payload %v vs link %v", payload.Created, link.Created)
		}

		if link.FileVersion != ShareLinkFileVersion(payload.FileVersion) {
			t.Fatalf("file_version = %q, want %q", link.FileVersion, payload.FileVersion)
		}

		if payload.Expiration == nil {
			if !link.Expiration.IsZero() {
				t.Fatalf("nil expiration decoded as %v, want zero", link.Expiration)
			}

			return
		}

		if !link.Expiration.Equal(*payload.Expiration) {
			t.Fatalf("expiration = %v, want %v", link.Expiration, *payload.Expiration)
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

		if slices.Contains(rawChecksums, got) {
			return
		}

		t.Fatalf("checksumFrom(\"\", ...) = %q, which no version carries (roots=%v checksums=%q)",
			got, rootFlags, rawChecksums)
	})
}

func FuzzClassifyTask(f *testing.F) {
	f.Add([]byte(`{"task_id":"t1","status":"success","result_data":{"document_id":42}}`))
	f.Add(
		[]byte(
			`{"task_id":"t2","status":"failure","result_data":{"duplicate_of":7,"duplicate_in_trash":true}}`,
		),
	)
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
				t.Fatalf(
					"document_id %d misclassified: %+v",
					*payload.ResultData.DocumentID,
					outcome,
				)
			}
		case payload.ResultData.DuplicateOf != nil:
			if !outcome.DuplicateRefused ||
				outcome.DocumentID != *payload.ResultData.DuplicateOf ||
				outcome.DuplicateInTrash != payload.ResultData.DuplicateInTrash {
				t.Fatalf(
					"duplicate_of %d misclassified: %+v",
					*payload.ResultData.DuplicateOf,
					outcome,
				)
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
