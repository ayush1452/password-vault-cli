package crypto

import (
	"fmt"
	"testing"
)

type deterministicReader struct {
	next byte
}

func (r *deterministicReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = r.next
		r.next++
	}
	return len(p), nil
}

func TestGeneratePasswordCharsets(t *testing.T) {
	reader := &deterministicReader{}
	SetRandomSource(reader)
	t.Cleanup(func() {
		SetRandomSource(nil)
	})

	tests := []struct {
		name    string
		charset Charset
		length  int
		allowed string
	}{
		{"alpha", CharsetAlpha, 16, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"},
		{"alnum", CharsetAlnum, 24, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"},
		{"alnum_special", CharsetAlnumSpecial, 32, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+[]{}<>?,.:;/'\"|\\~"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, err := GeneratePassword(tt.length, tt.charset)
			if err != nil {
				t.Fatalf("GeneratePassword() error = %v", err)
			}

			if got := len(pass); got != tt.length {
				t.Fatalf("GeneratePassword() length = %d, want %d", got, tt.length)
			}

			allowed := make(map[rune]struct{}, len(tt.allowed))
			for _, r := range tt.allowed {
				allowed[r] = struct{}{}
			}

			for _, r := range pass {
				if _, ok := allowed[r]; !ok {
					t.Fatalf("GeneratePassword() produced rune %q outside allowed set", r)
				}
			}
		})
	}
}

func TestGeneratePasswordInvalidInput(t *testing.T) {
	if _, err := GeneratePassword(0, CharsetAlpha); err == nil {
		t.Fatal("GeneratePassword() expected error for zero length")
	}

	if _, err := GeneratePassword(10, Charset("invalid")); err == nil {
		t.Fatal("GeneratePassword() expected error for invalid charset")
	}
}

func TestGenerateDiceware(t *testing.T) {
	reader := &deterministicReader{}
	SetRandomSource(reader)
	t.Cleanup(func() {
		SetRandomSource(nil)
	})

	words, err := GenerateDiceware(4)
	if err != nil {
		t.Fatalf("GenerateDiceware() error = %v", err)
	}

	if got := len(words); got != 4 {
		t.Fatalf("GenerateDiceware() length = %d, want 4", got)
	}

	for _, w := range words {
		if len(w) == 0 {
			t.Fatalf("GenerateDiceware() word %q is empty", w)
		}
	}
}

func TestGenerateDicewareInvalidInput(t *testing.T) {
	if _, err := GenerateDiceware(0); err == nil {
		t.Fatal("GenerateDiceware() expected error for zero word count")
	}
}

func BenchmarkGeneratePassword(b *testing.B) {
	SetRandomSource(nil)

	sizes := []int{16, 32, 64}
	for _, size := range sizes {
		b.Run(fmt.Sprintf("len=%d", size), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := GeneratePassword(size, CharsetAlnumSpecial); err != nil {
					b.Fatalf("GeneratePassword() error = %v", err)
				}
			}
		})
	}
}

// errorReader always returns an error
type errorReader struct{}

func (r *errorReader) Read(p []byte) (int, error) {
	return 0, fmt.Errorf("intentional read error")
}

// TestGeneratePasswordReadError tests error handling when random source fails
func TestGeneratePasswordReadError(t *testing.T) {
	SetRandomSource(&errorReader{})
	defer SetRandomSource(nil)

	_, err := GeneratePassword(10, CharsetAlpha)
	if err == nil {
		t.Error("GeneratePassword should fail when random source returns error")
	}
}

// TestGenerateDicewareReadError tests error handling when random source fails
func TestGenerateDicewareReadError(t *testing.T) {
	SetRandomSource(&errorReader{})
	defer SetRandomSource(nil)

	_, err := GenerateDiceware(5)
	if err == nil {
		t.Error("GenerateDiceware should fail when random source returns error")
	}
}

// TestSetRandomSourceNil tests resetting random source to default
func TestSetRandomSourceNil(t *testing.T) {
	// Set to custom reader
	custom := &deterministicReader{}
	SetRandomSource(custom)

	// Reset to nil (should use crypto/rand.Reader)
	SetRandomSource(nil)

	// Should work with default reader
	pass, err := GeneratePassword(10, CharsetAlpha)
	if err != nil {
		t.Fatalf("Expected no error after resetting to default: %v", err)
	}
	if len(pass) != 10 {
		t.Errorf("Expected length 10, got %d", len(pass))
	}
}

// TestGeneratePasswordAllCharsets tests all charset types
func TestGeneratePasswordAllCharsets(t *testing.T) {
	charsets := []Charset{
		CharsetAlpha,
		CharsetAlnum,
		CharsetAlnumSpecial,
		CharsetHex,
		CharsetNumeric,
		CharsetBase64,
	}

	for _, cs := range charsets {
		t.Run(string(cs), func(t *testing.T) {
			pass, err := GeneratePassword(20, cs)
			if err != nil {
				t.Fatalf("GeneratePassword(%s) error = %v", cs, err)
			}
			if len(pass) != 20 {
				t.Errorf("Expected length 20, got %d", len(pass))
			}
		})
	}
}

// TestGeneratePasswordNegativeLength tests negative length handling
func TestGeneratePasswordNegativeLength(t *testing.T) {
	_, err := GeneratePassword(-1, CharsetAlpha)
	if err == nil {
		t.Error("GeneratePassword should fail with negative length")
	}
}

// TestGenerateDicewareNegativeCount tests negative word count
func TestGenerateDicewareNegativeCount(t *testing.T) {
	_, err := GenerateDiceware(-5)
	if err == nil {
		t.Error("GenerateDiceware should fail with negative count")
	}
}

// TestGenerateDicewareLargeCount tests generating many diceware words
func TestGenerateDicewareLargeCount(t *testing.T) {
	words, err := GenerateDiceware(100)
	if err != nil {
		t.Fatalf("GenerateDiceware(100) error = %v", err)
	}
	if len(words) != 100 {
		t.Errorf("Expected 100 words, got %d", len(words))
	}
}

// mockRangeReader returns specific values to test different randomIndex branches
type mockRangeReader struct {
	values []byte
	index  int
}

func (r *mockRangeReader) Read(p []byte) (int, error) {
	if r.index >= len(r.values) {
		return 0, fmt.Errorf("no more values")
	}
	n := 0
	for i := range p {
		if r.index >= len(r.values) {
			break
		}
		p[i] = r.values[r.index]
		r.index++
		n++
	}
	if n == 0 {
		return 0, fmt.Errorf("no more values")
	}
	return n, nil
}

// TestRandomIndexMaxValZero tests randomIndex with non-positive maxVal
func TestRandomIndexMaxValZero(t *testing.T) {
	reader := &deterministicReader{}
	_, err := randomIndex(reader, 0)
	if err == nil {
		t.Error("randomIndex should fail with maxVal <= 0")
	}
}

// TestRandomIndexMaxValNegative tests randomIndex with negative maxVal
func TestRandomIndexMaxValNegative(t *testing.T) {
	reader := &deterministicReader{}
	_, err := randomIndex(reader, -5)
	if err == nil {
		t.Error("randomIndex should fail with negative maxVal")
	}
}

// TestRandomIndexMaxValOne tests randomIndex with maxVal == 1
func TestRandomIndexMaxValOne(t *testing.T) {
	reader := &deterministicReader{}
	result, err := randomIndex(reader, 1)
	if err != nil {
		t.Fatalf("randomIndex(1) error = %v", err)
	}
	if result != 0 {
		t.Errorf("Expected 0 for maxVal=1, got %d", result)
	}
}

// TestRandomIndexSmallRange tests ≤256 range (single byte)
func TestRandomIndexSmallRange(t *testing.T) {
	// Test range ≤ 256
	reader := &mockRangeReader{values: []byte{10, 50, 100}}
	for i := 0; i < 3; i++ {
		result, err := randomIndex(reader, 200)
		if err != nil {
			t.Fatalf("randomIndex(200) error = %v", err)
		}
		if result < 0 || result >= 200 {
			t.Errorf("randomIndex(200) = %d, out of range [0, 200)", result)
		}
	}
}

// TestRandomIndexMediumRange tests ≤65536 range (two bytes)
func TestRandomIndexMediumRange(t *testing.T) {
	// Test range > 256 and ≤ 65536
	reader := &mockRangeReader{values: []byte{0, 100, 1, 200}}
	for i := 0; i < 2; i++ {
		result, err := randomIndex(reader, 50000)
		if err != nil {
			t.Fatalf("randomIndex(50000) error = %v", err)
		}
		if result < 0 || result >= 50000 {
			t.Errorf("randomIndex(50000) = %d, out of range [0, 50000)", result)
		}
	}
}

// TestRandomIndexLargeRange tests > 65536 range (uint64)
func TestRandomIndexLargeRange(t *testing.T) {
	// Test range > 65536
	reader := &mockRangeReader{values: []byte{0, 0, 0, 0, 0, 0, 0, 100}}
	result, err := randomIndex(reader, 100000)
	if err != nil {
		t.Fatalf("randomIndex(100000) error = %v", err)
	}
	if result < 0 || result >= 100000 {
		t.Errorf("randomIndex(100000) = %d, out of range [0, 100000)", result)
	}
}

// TestRandomIndexReadFailure tests read error handling
func TestRandomIndexReadFailure(t *testing.T) {
	reader := &errorReader{}
	_, err := randomIndex(reader, 100)
	if err == nil {
		t.Error("randomIndex should fail when Read returns error")
	}
}

// TestDicewareWordsInitialization tests diceware word list initialization
func TestDicewareWordsInitialization(t *testing.T) {
	words := dicewareWords()
	if len(words) == 0 {
		t.Error("dicewareWords should not be empty")
	}

	// Should return same list on subsequent calls
	words2 := dicewareWords()
	if len(words) != len(words2) {
		t.Error("dicewareWords should return consistent length")
	}
}
