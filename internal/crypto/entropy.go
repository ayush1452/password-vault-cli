package crypto

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
)

// Charset represents a predefined set of characters that can be used for password generation.
// It's used to specify which character set to use when generating passwords.
type Charset string

const (
	// CharsetAlpha uses only alphabetic characters (a-z, A-Z)
	CharsetAlpha Charset = "alpha"
	// CharsetAlnum uses alphanumeric characters (a-z, A-Z, 0-9)
	CharsetAlnum Charset = "alnum"
	// CharsetAlnumSpecial uses alphanumeric and special characters
	CharsetAlnumSpecial Charset = "alnum_special"
	// CharsetHex uses hexadecimal characters (0-9, a-f)
	CharsetHex Charset = "hex"
	// CharsetNumeric uses only numeric characters (0-9)
	CharsetNumeric Charset = "numeric"
	// CharsetBase64 uses base64 URL-safe characters (A-Z, a-z, 0-9, -, _)
	CharsetBase64 Charset = "base64"
)

var (
	// defaultCharsets contains the default character sets for different character types
	defaultCharsets = map[Charset]string{
		CharsetAlpha:        "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ",
		CharsetAlnum:        "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
		CharsetAlnumSpecial: "!@#$%^&*()_+-=[]{}|;:,.<>?abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
		CharsetHex:          "0123456789abcdef",
		CharsetNumeric:      "0123456789",
		CharsetBase64:       "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_",
	}

	// randomSource is the source of random numbers
	randomSource     io.Reader = rand.Reader
	randomSourceLock sync.Mutex

	// dicewareList contains the list of diceware words
	dicewareList     []string
	dicewareListOnce sync.Once

	// dicewareList1 and dicewareList2 contain the diceware words
	dicewareList1 = []string{
		"a", "aaron", "abandoned", "aberdeen", "abilities", "ability", "able", "aboriginal", "abortion", "about",
		"above", "abraham", "abroad", "abs", "absence", "absent", "absolute", "absolutely", "absorption", "abstract",
	}

	dicewareList2 = []string{
		"academic", "academy", "acc", "accent", "accept", "acceptable", "acceptance", "accepted", "accepting", "accepts",
		"access", "accessed", "accessibility", "accessible", "accessing", "accessories", "accessory", "accident", "accidents", "accommodate",
	}
)

// SetRandomSource sets the random number generator source.
// If r is nil, it resets to the default crypto/rand.Reader.
func SetRandomSource(r io.Reader) {
	randomSourceLock.Lock()
	defer randomSourceLock.Unlock()

	if r == nil {
		randomSource = rand.Reader
	} else {
		randomSource = r
	}
}

// GeneratePassword generates a cryptographically secure random password with the specified length and character set.
// It returns the generated password or an error if the length is invalid or if there's a problem with the random source.
func GeneratePassword(length int, charset Charset) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("password length must be positive")
	}

	charSet, ok := defaultCharsets[charset]
	if !ok {
		return "", fmt.Errorf("invalid character set: %s", charset)
	}

	result := make([]byte, length)
	for i := 0; i < length; i++ {
		idx, err := randomIndex(randomSource, len(charSet))
		if err != nil {
			return "", fmt.Errorf("failed to generate random index: %w", err)
		}
		result[i] = charSet[idx]
	}

	return string(result), nil
}

// GenerateDiceware generates a list of random words using the diceware method.
// wordCount specifies how many words to generate.
// Returns a slice of words or an error if wordCount is invalid or if there's a problem with the random source.
func GenerateDiceware(wordCount int) ([]string, error) {
	if wordCount <= 0 {
		return nil, fmt.Errorf("word count must be positive")
	}

	words := dicewareWords()
	result := make([]string, 0, wordCount)

	for i := 0; i < wordCount; i++ {
		idx, err := randomIndex(randomSource, len(words))
		if err != nil {
			return nil, fmt.Errorf("failed to generate random index: %w", err)
		}
		result = append(result, words[idx])
	}

	return result, nil
}

func dicewareWords() []string {
	dicewareListOnce.Do(func() {
		merged := make([]string, 0, 8192) // Pre-allocate with approximate size
		merged = append(merged, dicewareList1...)
		merged = append(merged, dicewareList2...)
		dicewareList = merged
	})
	return dicewareList
}

// randomIndex generates a uniformly distributed random number in the range [0, maxVal-1]
// using rejection sampling to ensure uniform distribution and prevent modulo bias.
func randomIndex(r io.Reader, maxVal int) (int, error) {
	if maxVal <= 0 {
		return 0, fmt.Errorf("maxVal must be positive")
	}

	switch {
	// For small ranges (up to 256), use a single byte
	case maxVal <= 1:
		return 0, nil // Only one possible value
	case maxVal <= 256:
		var buf [1]byte
		usable := 256 - (256 % maxVal)
		for {
			if _, err := io.ReadFull(r, buf[:]); err != nil {
				return 0, fmt.Errorf("failed to read random byte: %w", err)
			}
			val := int(buf[0])
			if val < usable {
				return val % maxVal, nil
			}
		}

	// For medium ranges (up to 65536), use two bytes
	case maxVal <= 65536:
		var buf [2]byte
		usable := 65536 - (65536 % maxVal)
		for {
			if _, err := io.ReadFull(r, buf[:]); err != nil {
				return 0, fmt.Errorf("failed to read random bytes: %w", err)
			}
			val := int(binary.BigEndian.Uint16(buf[:]))
			if val < usable {
				return val % maxVal, nil
			}
		}

	// For larger ranges, use 8 bytes (uint64)
	default:
		var buf [8]byte
		// Calculate the maximum value that is a multiple of maxVal
		const maxUint64 = ^uint64(0)
		limit := maxUint64 - (maxUint64 % uint64(maxVal))

		for {
			// Read random bytes
			if _, err := io.ReadFull(r, buf[:]); err != nil {
				return 0, fmt.Errorf("failed to read random bytes: %w", err)
			}

			// Convert to uint64 in a safe way
			val := binary.BigEndian.Uint64(buf[:])

			// Use rejection sampling to ensure uniform distribution
			if val < limit {
				// Safe conversion since we know val < limit and limit is a multiple of maxVal
				result := val % uint64(maxVal)
				if result > uint64(^uint(0)>>1) {
					return 0, fmt.Errorf("random value %d overflows int", result)
				}
				return int(result), nil
			}
		}
	}
}
