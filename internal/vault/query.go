package vault

import (
	"strings"
	"unicode"

	"github.com/vault-cli/vault/internal/domain"
)

// ParseSearchTokens splits the raw search string into lower-cased tokens.
// Tokens are delimited by '+' or any whitespace character.
func ParseSearchTokens(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return unicode.IsSpace(r) || r == '+'
	})

	tokens := make([]string, 0, len(fields))
	for _, field := range fields {
		token := strings.TrimSpace(field)
		if token == "" {
			continue
		}
		tokens = append(tokens, strings.ToLower(token))
	}

	if len(tokens) == 0 {
		return nil
	}

	return tokens
}

// MatchesSearchTokens reports whether the entry satisfies all search tokens.
// Each token must be contained in at least one of name, username, url, or tags.
func MatchesSearchTokens(entry *domain.Entry, tokens []string) bool {
	if len(tokens) == 0 || entry == nil {
		return true
	}

	name := strings.ToLower(entry.Name)
	username := strings.ToLower(entry.Username)
	url := strings.ToLower(entry.URL)

	tags := make([]string, 0, len(entry.Tags))
	for _, tag := range entry.Tags {
		tags = append(tags, strings.ToLower(tag))
	}

	for _, token := range tokens {
		token = strings.ToLower(token)
		if token == "" {
			continue
		}

		if strings.Contains(name, token) ||
			strings.Contains(username, token) ||
			strings.Contains(url, token) ||
			tagContainsToken(tags, token) {
			continue
		}

		return false
	}

	return true
}

func tagContainsToken(tags []string, token string) bool {
	for _, tag := range tags {
		if strings.Contains(tag, token) {
			return true
		}
	}
	return false
}
