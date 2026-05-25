package notifications

import (
	"encoding/json"
	"testing"
)

func TestProgressParamsIncludesMetadata(t *testing.T) {
	params := ProgressParams{
		OperationID: 42,
		Status:      "success",
		Phase:       "done",
		Metadata:    json.RawMessage(`{"created":2,"requested":2,"failed":0}`),
	}

	raw, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal ProgressParams: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal ProgressParams: %v", err)
	}
	metadata, ok := got["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("metadata missing or wrong type: %#v", got["metadata"])
	}
	if metadata["created"] != float64(2) || metadata["requested"] != float64(2) || metadata["failed"] != float64(0) {
		t.Fatalf("metadata mismatch: %#v", metadata)
	}
}
