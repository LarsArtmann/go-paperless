package paperless

import (
	"encoding/json/v2"
	"testing"
)

func TestProbeNullOmitEmpty(t *testing.T) {
	zero := NewCorrespondentID(0)
	nonZero := NewCorrespondentID(7)

	must := func(name string, b []byte, err error) {
		t.Helper()

		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}

		t.Logf("%-24s %s", name, b)
	}

	b, err := json.Marshal(struct {
		C *CorrespondentID `json:"correspondent,omitempty"`
	}{&zero})
	must("zero ptr + omitempty", b, err)

	b, err = json.Marshal(struct {
		C *CorrespondentID `json:"correspondent,omitempty"`
	}{&nonZero})
	must("nonzero ptr + omitempty", b, err)

	b, err = json.Marshal(struct {
		C *CorrespondentID `json:"correspondent"`
	}{&zero})
	must("zero ptr no omitempty", b, err)
}
