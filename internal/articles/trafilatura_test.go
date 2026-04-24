package articles

import "testing"

func TestTrafilaturaExtractorRejectsEmptyURL(t *testing.T) {
	extractor := NewTrafilaturaExtractor()
	if _, err := extractor.Extract(""); err == nil {
		t.Fatal("expected error")
	}
}
