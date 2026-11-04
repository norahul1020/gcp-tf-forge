package hclgen

import (
	"fmt"
	"hash/crc32"
	"regexp"
	"strings"
	"unicode"
)

var (
	illegalCharRegex = regexp.MustCompile(`[^a-zA-Z0-9_-]`)
)

// Sanitizer cleans illegal characters from resource names and resolves identifier collisions.
type Sanitizer struct {
	seenIdentifiers map[string]string // sanitized -> original
}

// NewSanitizer creates a new identifier sanitizer instance.
func NewSanitizer() *Sanitizer {
	return &Sanitizer{
		seenIdentifiers: make(map[string]string),
	}
}

// Sanitize converts arbitrary cloud resource names into valid, unique Terraform identifiers.
func (s *Sanitizer) Sanitize(originalName string) string {
	if originalName == "" {
		return "unnamed_resource"
	}

	// 1. Replace illegal characters with underscores
	clean := illegalCharRegex.ReplaceAllString(originalName, "_")

	// 2. Terraform identifiers cannot start with numbers
	if unicode.IsDigit(rune(clean[0])) {
		clean = "res_" + clean
	}

	// 3. Lowercase standard
	clean = strings.ToLower(clean)

	// 4. Handle identifier collision (e.g., "my.app" and "my-app" both becoming "my_app")
	if orig, exists := s.seenIdentifiers[clean]; exists && orig != originalName {
		// Collision detected! Append deterministic 4-character CRC32 hash
		checksum := crc32.ChecksumIEEE([]byte(originalName))
		clean = fmt.Sprintf("%s_%04x", clean, checksum&0xffff)
	}

	s.seenIdentifiers[clean] = originalName
	return clean
}
