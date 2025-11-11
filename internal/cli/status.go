package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/vault-cli/vault/internal/store"
	"github.com/vault-cli/vault/internal/vault"
)

var statusJSON bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show vault status",
	Long:  "Display vault metadata, session state, and entry statistics.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatus()
	},
}

func init() {
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output status as JSON")
}

type statusInfo struct {
	VaultPath        string             `json:"vault_path"`
	Cipher           string             `json:"cipher"`
	KDF              vault.Argon2Params `json:"kdf"`
	SaltLength       int                `json:"salt_length"`
	MetadataCreated  string             `json:"metadata_created"`
	EntryCount       *int               `json:"entry_count,omitempty"`
	LastUpdated      *time.Time         `json:"last_updated,omitempty"`
	SessionState     string             `json:"session_state"`
	RemainingTTL     time.Duration      `json:"remaining_ttl"`
	RemainingTTLSecs int64              `json:"remaining_ttl_seconds"`
}

func runStatus() error {
	if cfg != nil {
		if vaultPath == "" {
			vaultPath = cfg.VaultPath
		}
		if profile == "" {
			profile = cfg.DefaultProfile
		}
	}

	if vaultPath == "" {
		return fmt.Errorf("vault path not configured")
	}

	unlocked := IsUnlocked()

	metaInfo, err := loadMetadataInfo()
	if err != nil {
		return err
	}

	var entryCount *int
	var lastUpdated *time.Time

	if unlocked {
		vaultStore := GetVaultStore()
		if vaultStore == nil {
			return fmt.Errorf("failed to access unlocked vault store")
		}
		defer CloseSessionStore()

		entries, err := vaultStore.ListEntries(profile, nil)
		if err != nil {
			return fmt.Errorf("failed to list entries: %w", err)
		}

		count := len(entries)
		entryCount = &count

		for _, entry := range entries {
			if entry == nil {
				continue
			}
			ts := entry.UpdatedAt
			if lastUpdated == nil || ts.After(*lastUpdated) {
				snapshot := ts
				lastUpdated = &snapshot
			}
		}
	}

	ttl := RemainingSessionTTL()
	sessionState := "locked"
	if unlocked {
		sessionState = "unlocked"
	}

	result := statusInfo{
		VaultPath:        vaultPath,
		Cipher:           metaInfo.Cipher,
		KDF:              metaInfo.KDF,
		SaltLength:       metaInfo.SaltLength,
		MetadataCreated:  metaInfo.MetadataCreated,
		EntryCount:       entryCount,
		LastUpdated:      lastUpdated,
		SessionState:     sessionState,
		RemainingTTL:     ttl,
		RemainingTTLSecs: int64(ttl.Seconds()),
	}

	if statusJSON {
		payload, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal status: %w", err)
		}
		if _, err := fmt.Println(string(payload)); err != nil {
			return fmt.Errorf("failed to write JSON output: %w", err)
		}
		return nil
	}

	// Use a helper function to handle fmt.Fprintf errors
	printStatus := func(format string, args ...interface{}) error {
		_, err := fmt.Fprintf(os.Stdout, format, args...)
		if err != nil {
			return fmt.Errorf("failed to write status: %w", err)
		}
		return nil
	}

	// Write each status line with error checking
	if err := printStatus("Vault: %s\n", result.VaultPath); err != nil {
		return err
	}
	if err := printStatus("Cipher: %s\n", result.Cipher); err != nil {
		return err
	}
	if err := printStatus("KDF: Argon2id (memory %d KB, iterations %d, parallelism %d, salt %d bytes)\n",
		result.KDF.Memory, result.KDF.Iterations, result.KDF.Parallelism, result.SaltLength); err != nil {
		return err
	}
	if err := printStatus("Created: %s\n", result.MetadataCreated); err != nil {
		return err
	}

	if unlocked {
		if err := printStatus("Entries: %d\n", *entryCount); err != nil {
			return err
		}
		if lastUpdated != nil {
			if err := printStatus("Last Updated: %s\n", lastUpdated.Format(time.RFC3339)); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintln(os.Stdout, "Last Updated: n/a"); err != nil {
				return fmt.Errorf("failed to write status: %w", err)
			}
		}
	} else {
		if _, err := fmt.Fprintln(os.Stdout, "Entries: (locked)"); err != nil {
			return fmt.Errorf("failed to write status: %w", err)
		}
	}

	if unlocked {
		if err := printStatus("Session: %s (expires in %s)\n", sessionState, ttl.Round(time.Second)); err != nil {
			return err
		}
	} else {
		if err := printStatus("Session: %s\n", sessionState); err != nil {
			return err
		}
	}

	return nil
}

func loadMetadataInfo() (*vault.MetadataInfo, error) {
	if IsUnlocked() {
		store := GetVaultStore()
		if store == nil {
			return nil, fmt.Errorf("failed to access unlocked vault store")
		}
		metadata, err := store.GetVaultMetadata()
		if err != nil {
			return nil, fmt.Errorf("failed to get vault metadata: %w", err)
		}
		info, _, err := vault.DecodeMetadataInfo(metadata)
		return info, err
	}

	boltStore := store.NewBoltStore()
	defer boltStore.CloseVault()

	dummyKey := make([]byte, vault.KeySize)
	if err := boltStore.OpenVault(vaultPath, dummyKey); err != nil {
		if errors.Is(err, store.ErrVaultLocked) {
			return nil, fmt.Errorf("vault is locked by another process")
		}
		return nil, fmt.Errorf("failed to open vault: %w", err)
	}

	metadata, err := boltStore.GetVaultMetadata()
	if err != nil {
		return nil, fmt.Errorf("failed to get vault metadata: %w", err)
	}

	info, _, err := vault.DecodeMetadataInfo(metadata)
	return info, err
}
