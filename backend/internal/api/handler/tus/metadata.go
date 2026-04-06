package tus

import (
	"encoding/base64"
	"strings"
)

// parseTusMetadata parses the Upload-Metadata header value as defined by the
// tus protocol: comma-separated key-value pairs where the value is
// base64-encoded. Example: "filename d29ybGQudHh0,upload_session_id YWJj".
func parseTusMetadata(header string) map[string]string {
	meta := make(map[string]string)
	if header == "" {
		return meta
	}
	for _, pair := range strings.Split(header, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, " ", 2)
		key := strings.TrimSpace(parts[0])
		if key == "" {
			continue
		}
		if len(parts) == 1 {
			meta[key] = ""
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(parts[1]))
		if err != nil {
			meta[key] = strings.TrimSpace(parts[1])
			continue
		}
		meta[key] = string(decoded)
	}
	return meta
}
