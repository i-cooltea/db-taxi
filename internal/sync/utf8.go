package sync

import "strings"

// valueForUTF8MB3Insert returns v suitable for inserting into a MySQL utf8/utf8mb3 column.
// String values are sanitized by removing 4-byte UTF-8 runes (e.g. emojis) to avoid Error 1366
// when the target column uses utf8 (3-byte) instead of utf8mb4.
func valueForUTF8MB3Insert(v interface{}) interface{} {
	if s, ok := v.(string); ok {
		return sanitizeStringForUTF8MB3(s)
	}
	return v
}

// sanitizeStringForUTF8MB3 removes runes that require 4 bytes in UTF-8 (e.g. emojis, U+10000 and above)
// so the string fits in MySQL utf8 (utf8mb3). Returns the original string if no such runes exist.
func sanitizeStringForUTF8MB3(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r < 0x10000 {
			b.WriteRune(r)
		}
		// skip supplementary plane runes (4-byte in UTF-8)
	}
	return b.String()
}
