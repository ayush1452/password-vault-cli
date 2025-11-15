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
