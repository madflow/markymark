package finder

import (
	"os"
	"strings"
)

// FindReadme looks for a README.md file (case-insensitive) in the current
// working directory and returns its name with original casing, or "" if not found.
func FindReadme() string {
	entries, err := os.ReadDir(".")
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.ToLower(entry.Name()) == "readme.md" {
			return entry.Name()
		}
	}

	return ""
}
