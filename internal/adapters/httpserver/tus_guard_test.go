package httpserver

import (
	"encoding/base64"
	"testing"
)

func TestParseTusMetadata(t *testing.T) {
	b64 := base64.StdEncoding.EncodeToString([]byte("r1"))
	b64name := base64.StdEncoding.EncodeToString([]byte("x.png"))
	h := "report_id " + b64 + ", filename " + b64name
	got := parseTusMetadata(h)
	if got["report_id"] != "r1" || got["filename"] != "x.png" {
		t.Fatalf("unexpected: %+v", got)
	}
}
