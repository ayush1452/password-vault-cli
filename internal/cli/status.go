package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vault-cli/vault/internal/config"
	"github.com/vault-cli/vault/internal/domain"
	"github.com/vault-cli/vault/internal/store"
	"github.com/vault-cli/vault/internal/vault"
)

var statusJSON bool

// NewStatus creates a new status command
func NewStatus(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show vault status",
		Long:  "Display vault metadata, session state, and entry statistics.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus()
		},
	}

	cmd.Flags().BoolVar(&statusJSON, "json", false, "Output status as JSON")
	return cmd
}

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

// StatusInfo captures the status information for the vault
type StatusInfo struct {
	VaultPath        string        `json:"vault_path"`
	Profile          string        `json:"profile,omitempty"`
	Cipher           string        `json:"cipher"`
	KDF              string        `json:"kdf"`
	SaltLength       int           `json:"salt_length"`
	MetadataCreated  string        `json:"metadata_created"`
	EntryCount       *int          `json:"entry_count,omitempty"`
	LastUpdated      *time.Time    `json:"last_updated,omitempty"`
	SessionState     string        `json:"session_state"`
	RemainingTTL     time.Duration `json:"remaining_ttl"`
	RemainingTTLSecs int64         `json:"remaining_ttl_seconds"`
}

func runStatus() error {
	log.Printf("Running status check with vault: %s, profile: %s", vaultPath, profile)

	// If no profile is specified, use "default" as a fallback
	if profile == "" {
		profile = "default"
		log.Printf("No profile specified, using default profile")
	}

	if cfg != nil {
		if vaultPath == "" {
			vaultPath = cfg.VaultPath
		}
		if profile == "" && cfg.DefaultProfile != "" {
			profile = cfg.DefaultProfile
			log.Printf("Using default profile from config: %s", profile)
		}
	}

	log.Printf("Final vault path: %s, profile: %s", vaultPath, profile)

	if vaultPath == "" {
		return fmt.Errorf("vault path not configured")
	}

	unlocked := IsUnlocked()

	metaInfo, err := loadMetadataInfo()
	if err != nil {
		log.Printf("Error loading metadata: %v", err)
		return fmt.Errorf("failed to load vault metadata: %w", err)
	}
	log.Printf("Successfully loaded metadata for vault")

	// Log basic metadata info
	if metaInfo != nil {
		log.Printf("Vault Cipher: %s, KDF: %+v, Salt Length: %d",
			metaInfo.Cipher,
			metaInfo.KDF,
			metaInfo.SaltLength)
	}

	// Initialize with default values
	var entryCount int
	var lastUpdated *time.Time

	if unlocked {
		vaultStore := GetVaultStore()
		if vaultStore == nil {
			log.Printf("Warning: failed to access unlocked vault store")
			// Continue with basic status
		} else {
			defer func() {
				if err := CloseSessionStore(); err != nil {
					log.Printf("Warning: failed to close session store: %v", err)
				}
			}()

			// Try to list profiles
			profiles, err := vaultStore.ListProfiles()
			if err != nil {
				log.Printf("Warning: failed to list profiles: %v", err)
			} else {
				// Look for the requested profile
				profileExists := false
				var profileObj *domain.Profile

				for _, p := range profiles {
					if p != nil && p.Name == profile {
						profileExists = true
						profileObj = p
						break
					}
				}

				if profileExists {
					log.Printf("Found profile: %+v", profileObj)

					// Try to list entries for the profile
					entries, err := vaultStore.ListEntries(profile, nil)
					if err != nil {
						log.Printf("Warning: failed to list entries for profile '%s': %v", profile, err)
					} else {
						totalEntries := len(entries)
						entryCount = totalEntries
						log.Printf("Found %d entries in profile '%s'", totalEntries, profile)

						// Find the most recent update time
						if totalEntries > 0 {
							var latestTime time.Time
							for _, entry := range entries {
								if entry != nil && (latestTime.IsZero() || entry.UpdatedAt.After(latestTime)) {
									latestTime = entry.UpdatedAt
								}
							}
							if !latestTime.IsZero() {
								lastUpdated = &latestTime
							}
						}
					}
				} else {
					log.Printf("Profile '%s' not found in vault, showing basic status only", profile)
				}
			}
		}
	}

	sessionState := "locked"
	if unlocked {
		sessionState = "unlocked"
	}

	ttl := RemainingSessionTTL()

	// Format KDF as string
	kdfStr := fmt.Sprintf("Argon2id (memory %d KB, iterations %d, parallelism %d)",
		metaInfo.KDF.Memory, metaInfo.KDF.Iterations, metaInfo.KDF.Parallelism)

	// Prepare status output
	status := &StatusInfo{
		VaultPath:        vaultPath,
		Profile:          profile,
		Cipher:           metaInfo.Cipher,
		KDF:              kdfStr,
		SaltLength:       metaInfo.SaltLength,
		MetadataCreated:  metaInfo.MetadataCreated,
		SessionState:     sessionState,
		RemainingTTL:     ttl,
		RemainingTTLSecs: int64(ttl.Seconds()),
	}

	// Only set EntryCount if we have a valid value
	if entryCount > 0 {
		tempCount := entryCount // Create a new variable to hold the value
		status.EntryCount = &tempCount
	}

	// LastUpdated is already a pointer, so we can assign it directly
	status.LastUpdated = lastUpdated

	if statusJSON {
		payload, err := json.MarshalIndent(status, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal status: %w", err)
		}
		if _, err := fmt.Println(string(payload)); err != nil {
			return fmt.Errorf("failed to write JSON output: %w", err)
		}
		return nil
	}

	// Format the output for non-JSON
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Vault: %s\n", status.VaultPath))
	if status.Profile != "" {
		result.WriteString(fmt.Sprintf("Profile: %s\n", status.Profile))
	}
	result.WriteString(fmt.Sprintf("Cipher: %s\n", status.Cipher))
	result.WriteString(fmt.Sprintf("KDF: %s\n", status.KDF))
	result.WriteString(fmt.Sprintf("Created: %s\n", status.MetadataCreated))
	result.WriteString(fmt.Sprintf("Session: %s\n", status.SessionState))

	if status.EntryCount != nil {
		result.WriteString(fmt.Sprintf("Entries: %d\n", *status.EntryCount))
	}

	if status.LastUpdated != nil {
		result.WriteString(fmt.Sprintf("Last Updated: %s\n", status.LastUpdated.Format(time.RFC3339)))
	}

	if status.RemainingTTL > 0 {
		result.WriteString(fmt.Sprintf("Session TTL: %s\n", status.RemainingTTL.Round(time.Second)))
	}

	// Print the result
	fmt.Print(result.String())

	return nil
}


func loadMetadataInfo() (*vault.MetadataInfo, error) {
	if IsUnlocked() {
		vaultStore := GetVaultStore()
		if vaultStore == nil {
			return nil, fmt.Errorf("failed to access unlocked vault store")
		}
		metadata, err := vaultStore.GetVaultMetadata()
		if err != nil {
			return nil, fmt.Errorf("failed to get vault metadata: %w", err)
		}
		info, _, err := vault.DecodeMetadataInfo(metadata)
		return info, err
	}

	boltStore := store.NewBoltStore()
	defer func() {
		if err := boltStore.CloseVault(); err != nil {
			log.Printf("Warning: failed to close bolt store: %v", err)
		}
	}()

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
