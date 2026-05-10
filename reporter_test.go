package gaugo

import (
	"strings"
	"testing"
)

func TestMetricFailureMetadataOmitsRawDetails(t *testing.T) {
	t.Parallel()

	m := MetricResult{
		Name:    "AnswerRelevancy",
		Pass:    false,
		Reason:  "bad",
		Details: []byte(`{"secret":"token-123","raw":"sensitive"}`),
	}

	metadata := metricFailureMetadata(m)
	if metadata != "details_bytes=40" {
		t.Fatalf("metadata got=%q want=%q", metadata, "details_bytes=40")
	}
	if strings.Contains(metadata, "token-123") || strings.Contains(metadata, "secret") {
		t.Fatalf("metadata should not include raw details: %q", metadata)
	}
}

func TestMetricFailureMetadataHandlesBinaryDetails(t *testing.T) {
	t.Parallel()

	m := MetricResult{
		Name:    "M1",
		Pass:    false,
		Score:   0,
		Reason:  "failed",
		Details: []byte{0xff, 0xfe, 0x00},
	}

	metadata := metricFailureMetadata(m)
	if metadata != "details_bytes=3" {
		t.Fatalf("metadata got=%q want=%q", metadata, "details_bytes=3")
	}
}
