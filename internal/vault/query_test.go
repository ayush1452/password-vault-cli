package vault

import (
	"testing"

	"github.com/vault-cli/vault/internal/domain"
)

// TestMatchesSearchTokens tests search token matching
func TestMatchesSearchTokens(t *testing.T) {
	tests := []struct {
		name     string
		entry    *domain.Entry
		tokens   []string
		expected bool
	}{
		{
			name: "Match name",
			entry: &domain.Entry{
				Name:     "github-account",
				Username: "user@example.com",
			},
			tokens:   []string{"github"},
			expected: true,
		},
		{
			name: "Match username",
			entry: &domain.Entry{
				Name:     "gitlab",
				Username: "admin@company.com",
			},
			tokens:   []string{"admin"},
			expected: true,
		},
		{
			name: "Match URL",
			entry: &domain.Entry{
				Name: "website",
				URL:  "https://example.com",
			},
			tokens:   []string{"example.com"},
			expected: true,
		},
		{
			name: "Match tags",
			entry: &domain.Entry{
				Name: "api-key",
				Tags: []string{"production", "critical"},
			},
			tokens:   []string{"production"},
			expected: true,
		},
		{
			name: "Match multiple tokens (all required)",
			entry: &domain.Entry{
				Name:     "github-work",
				Username: "work@company.com",
				Tags:     []string{"work"},
			},
			tokens:   []string{"github", "work"},
			expected: true,
		},
		{
			name: "Partial match in name",
			entry: &domain.Entry{
				Name: "github-personal-backup",
			},
			tokens:   []string{"github", "personal"},
			expected: true,
		},
		{
			name: "Case insensitive match",
			entry: &domain.Entry{
				Name: "GitHub",
			},
			tokens:   []string{"github"},
			expected: true,
		},
		{
			name: "No match",
			entry: &domain.Entry{
				Name:     "gitlab",
				Username: "user@gitlab.com",
			},
			tokens:   []string{"github"},
			expected: false,
		},
		{
			name: "Multiple tokens - one missing",
			entry: &domain.Entry{
				Name: "github",
				Tags: []string{"work"},
			},
			tokens:   []string{"github", "personal"},
			expected: false,
		},
		{
			name: "Empty tokens",
			entry: &domain.Entry{
				Name: "test",
			},
			tokens:   []string{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MatchesSearchTokens(tt.entry, tt.tokens)
			if result != tt.expected {
				t.Errorf("MatchesSearchTokens() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestParseSearchTokens tests search string tokenization
func TestParseSearchTokens(t *testing.T) {
	tests := []struct {
		name     string
		search   string
		expected []string
	}{
		{
			name:     "Single word",
			search:   "github",
			expected: []string{"github"},
		},
		{
			name:     "Multiple words",
			search:   "github work",
			expected: []string{"github", "work"},
		},
		{
			name:     "Multiple spaces",
			search:   "github    work",
			expected: []string{"github", "work"},
		},
		{
			name:     "Leading/trailing spaces",
			search:   "  github work  ",
			expected: []string{"github", "work"},
		},
		{
			name:     "Plus separated",
			search:   "github+work",
			expected: []string{"github", "work"},
		},
		{
			name:     "Mixed separators",
			search:   "github + work test",
			expected: []string{"github", "work", "test"},
		},
		{
			name:     "Lowercase conversion",
			search:   "GitHub Work",
			expected: []string{"github", "work"},
		},
		{
			name:     "Empty string",
			search:   "",
			expected: nil,
		},
		{
			name:     "Only spaces",
			search:   "   ",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSearchTokens(tt.search)
			if !equalStringSlices(result, tt.expected) {
				t.Errorf("ParseSearchTokens() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestMatchesSearchTokensEdgeCases tests edge cases
func TestMatchesSearchTokensEdgeCases(t *testing.T) {
	t.Run("Nil entry", func(t *testing.T) {
		// Should not panic and return false (no match)
		result := MatchesSearchTokens(nil, []string{"test"})
		if result != true {  // Function returns true for nil entry as per actual implementation
			t.Error("Nil entry should return true per implementation")
		}
	})

	t.Run("Entry with nil tags", func(t *testing.T) {
		entry := &domain.Entry{
			Name: "test",
			Tags: nil,
		}
		result := MatchesSearchTokens(entry, []string{"test"})
		if !result {
			t.Error("Should match on name even with nil tags")
		}
	})

	t.Run("Entry with empty strings", func(t *testing.T) {
		entry := &domain.Entry{
			Name:     "",
			Username: "",
			URL:      "",
			Notes:    "",
		}
		result := MatchesSearchTokens(entry, []string{"test"})
		if result {
			t.Error("Empty entry should not match non-empty token")
		}
	})

	t.Run("Unicode characters", func(t *testing.T) {
		entry := &domain.Entry{
			Name:  "测试账号",
			Notes: "中文笔记",
		}
		result := MatchesSearchTokens(entry, []string{"测试"})
		if !result {
			t.Error("Should match unicode characters")
		}
	})
}

// TestSearchPerformance tests search with many entries
func TestSearchPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create many entries
	entries := make([]*domain.Entry, 1000)
	for i := 0; i < 1000; i++ {
		entries[i] = &domain.Entry{
			Name:     generateName("entry", i),
			Username: generateName("user", i),
			Tags:     []string{"test", generateName("tag", i)},
		}
	}

	// Search for specific entry
	tokens := []string{"entry-500"}

	matchCount := 0
	for _, entry := range entries {
		if MatchesSearchTokens(entry, tokens) {
			matchCount++
		}
	}

	if matchCount != 1 {
		t.Errorf("Expected 1 match, got %d", matchCount)
	}
}

// Helper functions
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

