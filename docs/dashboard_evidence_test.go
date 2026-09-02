package docs_test

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDashboardEvidenceCapturesHaveReviewedViewports(t *testing.T) {
	_, here, _, _ := runtime.Caller(0)
	assets := filepath.Join(filepath.Dir(here), "assets")
	for _, tc := range []struct {
		name          string
		width, height uint32
	}{
		{name: "dashboard-analytics.png", width: 1280, height: 2600},
		{name: "dashboard-attention.png", width: 1280, height: 1700},
		{name: "dashboard-analytics-mobile.png", width: 390, height: 844},
		{name: "dashboard-attention-mobile.png", width: 390, height: 844},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, err := os.ReadFile(filepath.Join(assets, tc.name))
			if err != nil {
				t.Fatal(err)
			}
			if len(body) < 24 || string(body[:8]) != "\x89PNG\r\n\x1a\n" {
				t.Fatalf("%s is not a complete PNG capture", tc.name)
			}
			width := binary.BigEndian.Uint32(body[16:20])
			height := binary.BigEndian.Uint32(body[20:24])
			if width != tc.width || height != tc.height {
				t.Fatalf("%s dimensions = %dx%d, want %dx%d", tc.name, width, height, tc.width, tc.height)
			}
		})
	}
}
