package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vault-cli/vault/internal/config"
	internalcrypto "github.com/vault-cli/vault/internal/crypto"
)

type clipboardSpy struct {
	called bool
	secret string
	ttl    time.Duration
	err    error
}

func (c *clipboardSpy) copy(text string, ttl time.Duration) error {
	c.called = true
	c.secret = text
	c.ttl = ttl
	return c.err
}

type testReader struct {
	next byte
}

func (r *testReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = r.next
		r.next++
	}
	return len(p), nil
}

func TestPassgenGeneratePassword(t *testing.T) {
	reader := &testReader{}
	internalcrypto.SetRandomSource(reader)
	t.Cleanup(func() { internalcrypto.SetRandomSource(nil) })

	cmd := NewPassgenCommand(&config.Config{})
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"--length", "24", "--charset", "alnum"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	if len(output) != 24 {
		t.Fatalf("expected password length 24, got %d", len(output))
	}

	allowed := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for _, r := range output {
		if !strings.ContainsRune(allowed, r) {
			t.Fatalf("character %q not allowed for charset alnum", r)
		}
	}
}

func TestPassgenDicewareMutualExclusion(t *testing.T) {
	cmd := NewPassgenCommand(&config.Config{})
	cmd.SetArgs([]string{"--words", "4", "--length", "16"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error when using --words with --length")
	}
}

func TestPassgenCopyToClipboard(t *testing.T) {
	reader := &testReader{}
	internalcrypto.SetRandomSource(reader)
	t.Cleanup(func() { internalcrypto.SetRandomSource(nil) })

	s := &clipboardSpy{}
	originalCopy := copyToClipboard
	originalAvailable := clipboardIsAvailable
	copyToClipboard = s.copy
	clipboardIsAvailable = func() bool { return true }
	t.Cleanup(func() {
		copyToClipboard = originalCopy
		clipboardIsAvailable = originalAvailable
	})

	cfg := &config.Config{ClipboardTTL: 45 * time.Second}
	cmd := NewPassgenCommand(cfg)
	stdout := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"--copy", "--ttl", "5"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !s.called {
		t.Fatal("expected clipboard copy to be invoked")
	}

	if s.ttl != 5*time.Second {
		t.Fatalf("expected ttl 5s, got %v", s.ttl)
	}

	if strings.TrimSpace(stdout.String()) == "" {
		t.Fatal("expected confirmation output when copying")
	}
}

func TestPassgenCopyClipboardUnavailable(t *testing.T) {
	reader := &testReader{}
	internalcrypto.SetRandomSource(reader)
	t.Cleanup(func() { internalcrypto.SetRandomSource(nil) })

	originalCopy := copyToClipboard
	originalAvailable := clipboardIsAvailable
	copyToClipboard = func(string, time.Duration) error { return nil }
	clipboardIsAvailable = func() bool { return false }
	t.Cleanup(func() {
		copyToClipboard = originalCopy
		clipboardIsAvailable = originalAvailable
	})

	cmd := NewPassgenCommand(&config.Config{})
	cmd.SetArgs([]string{"--copy"})

	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error when clipboard is unavailable")
	}
}

func TestResolveClipboardTTL(t *testing.T) {
	t.Run("negative override invalid", func(t *testing.T) {
		if _, err := resolveClipboardTTL(-2, nil); err == nil {
			t.Fatal("expected error for negative ttl override")
		}
	})

	t.Run("explicit override", func(t *testing.T) {
		ttl, err := resolveClipboardTTL(10, nil)
		if err != nil {
			t.Fatalf("resolveClipboardTTL() error = %v", err)
		}
		if ttl != 10*time.Second {
			t.Fatalf("expected 10s ttl, got %v", ttl)
		}
	})

	t.Run("config default", func(t *testing.T) {
		cfg := &config.Config{ClipboardTTL: 12 * time.Second}
		ttl, err := resolveClipboardTTL(-1, cfg)
		if err != nil {
			t.Fatalf("resolveClipboardTTL() error = %v", err)
		}
		if ttl != 12*time.Second {
			t.Fatalf("expected config ttl, got %v", ttl)
		}
	})

	t.Run("fallback default", func(t *testing.T) {
		ttl, err := resolveClipboardTTL(-1, nil)
		if err != nil {
			t.Fatalf("resolveClipboardTTL() error = %v", err)
		}
		if ttl != 30*time.Second {
			t.Fatalf("expected fallback 30s ttl, got %v", ttl)
		}
	})
}

func TestPassgenCopyPropagatesClipboardError(t *testing.T) {
	reader := &testReader{}
	internalcrypto.SetRandomSource(reader)
	t.Cleanup(func() { internalcrypto.SetRandomSource(nil) })

	errClipboard := errors.New("clipboard error")
	s := &clipboardSpy{err: errClipboard}
	originalCopy := copyToClipboard
	originalAvailable := clipboardIsAvailable
	copyToClipboard = s.copy
	clipboardIsAvailable = func() bool { return true }
	t.Cleanup(func() {
		copyToClipboard = originalCopy
		clipboardIsAvailable = originalAvailable
	})

	cmd := NewPassgenCommand(&config.Config{})
	cmd.SetArgs([]string{"--copy"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "clipboard") {
		t.Fatalf("expected clipboard error, got %v", err)
	}
}
