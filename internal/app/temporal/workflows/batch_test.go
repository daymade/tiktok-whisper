package workflows

import (
	"testing"
	"time"
)

func TestBuildBatchFileIndexPreservesOriginalOrdering(t *testing.T) {
	files := []BatchFile{
		{FileID: "f1"},
		{FileID: "f2"},
		{FileID: "f3"},
	}

	index := buildBatchFileIndex(files)

	if index["f1"] != 0 || index["f2"] != 1 || index["f3"] != 2 {
		t.Fatalf("unexpected index map: %#v", index)
	}
}

func TestMergeBatchRetryResultsReplacesFailedEntries(t *testing.T) {
	first := BatchWorkflowResult{
		BatchID:      "batch-1",
		TotalFiles:   3,
		SuccessCount: 1,
		FailureCount: 2,
		Results: []SingleFileWorkflowResult{
			{FileID: "f1", Provider: "a"},
			{FileID: "f2", Error: "boom"},
			{FileID: "f3", Error: "bad"},
		},
		ProcessingTime: 2 * time.Second,
	}
	retry := BatchWorkflowResult{
		BatchID:      "batch-1",
		TotalFiles:   2,
		SuccessCount: 1,
		FailureCount: 1,
		Results: []SingleFileWorkflowResult{
			{FileID: "f2", Provider: "b"},
			{FileID: "f3", Error: "still bad"},
		},
		ProcessingTime: 3 * time.Second,
	}

	merged := mergeBatchRetryResults(first, retry)

	if merged.TotalFiles != 3 {
		t.Fatalf("expected total files to stay 3, got %d", merged.TotalFiles)
	}
	if len(merged.Results) != 3 {
		t.Fatalf("expected 3 merged results, got %d", len(merged.Results))
	}
	if merged.Results[1].Provider != "b" || merged.Results[1].Error != "" {
		t.Fatalf("expected retry result to replace failed slot for f2, got %#v", merged.Results[1])
	}
	if merged.Results[2].Error == "" {
		t.Fatalf("expected failed retry result for f3 to remain failed, got %#v", merged.Results[2])
	}
	if merged.SuccessCount != 2 || merged.FailureCount != 1 {
		t.Fatalf("unexpected success/failure counts: %d/%d", merged.SuccessCount, merged.FailureCount)
	}
}
