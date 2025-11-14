package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vault-cli/vault/internal/clipboard"
	"github.com/vault-cli/vault/internal/config"
	internalcrypto "github.com/vault-cli/vault/internal/crypto"
)

var (
	copyToClipboard      = clipboard.CopyWithTimeout
	clipboardIsAvailable = clipboard.IsAvailable
)

type passgenOptions struct {
	length  int
	words   int
	charset string
	copy    bool
	ttl     int
}

var passgenCmd = newPassgenCommand(nil)

func newPassgenCommand(conf *config.Config) *cobra.Command {
	opts := &passgenOptions{
		length:  20,
		charset: string(internalcrypto.CharsetAlnumSpecial),
		ttl:     -1,
	}

	cmd := &cobra.Command{
		Use:   "passgen",
		Short: "Generate secure passwords or passphrases",
		Long: `Generate secure passwords using configurable character sets or
Diceware-style passphrases, with optional clipboard support.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPassgen(cmd, opts, conf)
		},
	}

	cmd.Flags().IntVar(&opts.length, "length", opts.length, "Length of generated password (characters)")
	cmd.Flags().IntVar(&opts.words, "words", 0, "Number of words for Diceware passphrase")
	cmd.Flags().BoolVar(&opts.copy, "copy", false, "Copy the generated value to the clipboard")
	cmd.Flags().IntVar(&opts.ttl, "ttl", opts.ttl, "Clipboard clear timeout in seconds (-1 to use config default)")
	cmd.Flags().StringVar(&opts.charset, "charset", opts.charset, "Character set (alpha|alnum|alnum_special)")

	return cmd
}

// NewPassgenCommand creates a passgen command for testing.
func NewPassgenCommand(conf *config.Config) *cobra.Command {
	return newPassgenCommand(conf)
}

func runPassgen(cmd *cobra.Command, opts *passgenOptions, conf *config.Config) error {
	if conf == nil {
		conf = cfg
	}

	useWords := opts.words > 0

	if useWords {
		if cmd.Flags().Changed("length") {
			return fmt.Errorf("--words cannot be used with --length")
		}
		if cmd.Flags().Changed("charset") {
			return fmt.Errorf("--words cannot be used with --charset")
		}
		if opts.words <= 0 {
			return fmt.Errorf("--words must be positive")
		}

		words, err := internalcrypto.GenerateDiceware(opts.words)
		if err != nil {
			return fmt.Errorf("failed to generate passphrase: %w", err)
		}

		output := strings.Join(words, " ")
		return outputPassgen(cmd, output, opts, conf)
	}

	charset := internalcrypto.Charset(strings.ToLower(opts.charset))
	switch charset {
	case internalcrypto.CharsetAlpha, internalcrypto.CharsetAlnum, internalcrypto.CharsetAlnumSpecial:
	default:
		return fmt.Errorf("invalid charset: %s (valid: alpha, alnum, alnum_special)", opts.charset)
	}

	if opts.length <= 0 {
		return fmt.Errorf("--length must be positive")
	}

	password, err := internalcrypto.GeneratePassword(opts.length, charset)
	if err != nil {
		return fmt.Errorf("failed to generate password: %w", err)
	}

	return outputPassgen(cmd, password, opts, conf)
}

func outputPassgen(cmd *cobra.Command, secret string, opts *passgenOptions, conf *config.Config) error {
	// Get the output writer
	out := cmd.OutOrStdout()

	// Use the shared writeOutput function for consistent error handling
	writeOutput := func(format string, args ...interface{}) error {
		_, err := fmt.Fprintf(out, format, args...)
		if err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		return nil
	}

	if !opts.copy {
		if err := writeOutput("%s\n", secret); err != nil {
			return fmt.Errorf("failed to write password: %w", err)
		}
		return nil
	}

	if !clipboardIsAvailable() {
		return fmt.Errorf("clipboard not available, remove --copy to print instead")
	}

	ttl, err := resolveClipboardTTL(opts.ttl, conf)
	if err != nil {
		return err
	}

	if err := copyToClipboard(secret, ttl); err != nil {
		return fmt.Errorf("failed to copy to clipboard: %w", err)
	}

	if err := writeOutput("✓ Password copied to clipboard (clears in %s)\n", ttl.Round(time.Second)); err != nil {
		return fmt.Errorf("failed to write success message: %w", err)
	}
	return nil
}

func resolveClipboardTTL(override int, conf *config.Config) (time.Duration, error) {
	if override < -1 {
		return 0, fmt.Errorf("--ttl must be -1 (config default) or a non-negative number of seconds")
	}

	if override >= 0 {
		return time.Duration(override) * time.Second, nil
	}

	if conf != nil && conf.ClipboardTTL > 0 {
		return conf.ClipboardTTL, nil
	}

	return 30 * time.Second, nil
}
