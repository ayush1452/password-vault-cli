package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Perform security and health checks",
	Long: `Perform comprehensive security and health checks on the vault.

This command checks:
- File permissions and ownership
- Vault integrity and structure
- KDF parameter strength
- Configuration security
- System security recommendations

Example:
  vault doctor`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDoctor()
	},
}

func runDoctor() error {
	// Helper function to write output and check for errors
	printStatus := func(format string, args ...interface{}) error {
		_, err := fmt.Fprintf(os.Stdout, format, args...)
		if err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		return nil
	}

	if err := printStatus("Vault Security & Health Check\n"); err != nil {
		return err
	}
	if err := printStatus("=============================\n"); err != nil {
		return err
	}

	issues := 0
	warnings := 0

	// Check 1: Vault file existence and permissions
	if err := printStatus("\n1. Vault File Security\n"); err != nil {
		return err
	}

	if _, err := os.Stat(vaultPath); os.IsNotExist(err) {
		if err := printStatus("   ❌ Vault file not found: %s\n", vaultPath); err != nil {
			return err
		}
		issues++
	} else {
		info, err := os.Stat(vaultPath)
		if err != nil {
			if err := printStatus("   ❌ Cannot check vault file: %v\n", err); err != nil {
				return err
			}
			issues++
		} else {
			perm := info.Mode().Perm()
			switch {
			case perm == 0o600:
				if err := printStatus("   ✅ Vault file permissions: %o (secure)\n", perm); err != nil {
					return err
				}
			case perm&0o077 != 0:
				if err := printStatus("   ❌ Vault file permissions: %o (too permissive, should be 0600)\n", perm); err != nil {
					return err
				}
				if err := printStatus("      Fix with: chmod 600 %s\n", vaultPath); err != nil {
					return err
				}
				issues++
			default:
				if err := printStatus("   ⚠️  Vault file permissions: %o (acceptable but 0600 recommended)\n", perm); err != nil {
					return err
				}
				warnings++
			}
		}
	}

	// Check 2: Configuration file security
	if err := printStatus("\n2. Configuration Security\n"); err != nil {
		return err
	}
	if cfgFile != "" {
		if info, err := os.Stat(cfgFile); err == nil {
			perm := info.Mode().Perm()
			switch {
			case perm == 0o600:
				if err := printStatus("   ✅ Config file permissions: %o (secure)\n", perm); err != nil {
					return err
				}
			case perm&0o077 != 0:
				if err := printStatus("   ❌ Config file permissions: %o (too permissive, should be 0600)\n", perm); err != nil {
					return err
				}
				if err := printStatus("      Fix with: chmod 600 %s\n", cfgFile); err != nil {
					return err
				}
				issues++
			default:
				if err := printStatus("   ⚠️  Config file permissions: %o (acceptable but 0600 recommended)\n", perm); err != nil {
					return err
				}
				warnings++
			}
		} else {
			if err := printStatus("   ✅ Config file not found (using defaults)\n"); err != nil {
				return err
			}
		}
	}

	// Check 3: Directory permissions
	if err := printStatus("\n3. Directory Security\n"); err != nil {
		return err
	}
	vaultDir := filepath.Dir(vaultPath)
	if info, err := os.Stat(vaultDir); err == nil {
		perm := info.Mode().Perm()
		if perm&0o077 == 0 {
			if err := printStatus("   ✅ Vault directory permissions: %o (secure)\n", perm); err != nil {
				return err
			}
		} else {
			if err := printStatus("   ⚠️  Vault directory permissions: %o (consider 0700 for better security)\n", perm); err != nil {
				return err
			}
			warnings++
		}
	}

	// Check 4: Vault integrity (if unlocked)
	if err := printStatus("\n4. Vault Integrity\n"); err != nil {
		return err
	}
	if IsUnlocked() {
		vaultStore := GetVaultStore()
		if err := vaultStore.VerifyIntegrity(); err != nil {
			if err := printStatus("   ❌ Vault integrity check failed: %v\n", err); err != nil {
				return err
			}
			issues++
		} else {
			if err := printStatus("   ✅ Vault structure is valid\n"); err != nil {
				return err
			}
		}

		// Check metadata
		metadata, err := vaultStore.GetVaultMetadata()
		if err != nil {
			if err := printStatus("   ❌ Cannot read vault metadata: %v\n", err); err != nil {
				return err
			}
			issues++
		} else {
			if err := printStatus("   ✅ Vault version: %s\n", metadata.Version); err != nil {
				return err
			}
			if err := printStatus("   ✅ Created: %s\n", metadata.CreatedAt.Format("2006-01-02 15:04:05")); err != nil {
				return err
			}
		}
	} else {
		if err := printStatus("   ⚠️  Vault is locked, cannot perform integrity checks\n"); err != nil {
			return err
		}
		if err := printStatus("      Run 'vault unlock' first for complete check\n"); err != nil {
			return err
		}
		warnings++
	}

	// Check 5: KDF parameters (if available)
	if err := printStatus("\n5. Cryptographic Parameters\n"); err != nil {
		return err
	}
	if IsUnlocked() {
		vaultStore := GetVaultStore()
		metadata, err := vaultStore.GetVaultMetadata()
		if err == nil && metadata.KDFParams != nil {
			if memory, ok := metadata.KDFParams["memory"].(uint32); ok {
				switch {
				case memory >= 65536: // 64 MB
					if err := printStatus("   ✅ KDF memory parameter: %d KB (strong)\n", memory); err != nil {
						return err
					}
				case memory >= 8192: // 8 MB
					if err := printStatus("   ⚠️  KDF memory parameter: %d KB (acceptable but consider increasing)\n", memory); err != nil {
						return err
					}
					warnings++
				default:
					if err := printStatus("   ❌ KDF memory parameter: %d KB (weak, should be at least 8192 KB)\n", memory); err != nil {
						return err
					}
					issues++
				}
			}

			if iterations, ok := metadata.KDFParams["iterations"].(uint32); ok {
				if iterations >= 3 {
					if err := printStatus("   ✅ KDF iterations: %d (adequate)\n", iterations); err != nil {
						return err
					}
				} else {
					if err := printStatus("   ⚠️  KDF iterations: %d (consider increasing for better security)\n", iterations); err != nil {
						return err
					}
					warnings++
				}
			}

			if parallelism, ok := metadata.KDFParams["parallelism"].(uint8); ok {
				if parallelism >= 2 {
					if err := printStatus("   ✅ KDF parallelism: %d (good)\n", parallelism); err != nil {
						return err
					}
				} else {
					if err := printStatus("   ⚠️  KDF parallelism: %d (consider increasing for better performance)\n", parallelism); err != nil {
						return err
					}
					warnings++
				}
			}
		}
	}

	// Check 6: System security recommendations
	if err := printStatus("\n6. System Security Recommendations\n"); err != nil {
		return err
	}

	// Check for swap files
	if _, err := os.Stat("/proc/swaps"); err == nil {
		if err := printStatus("   ⚠️  Swap files detected - secrets may be written to disk\n"); err != nil {
			return err
		}
		if err := printStatus("      Consider disabling swap or using encrypted swap\n"); err != nil {
			return err
		}
		warnings++
	}

	// Check clipboard timeout
	if cfg.ClipboardTTL > 60*time.Second {
		if err := printStatus("   ⚠️  Clipboard timeout is %v (consider reducing for better security)\n", cfg.ClipboardTTL); err != nil {
			return err
		}
		warnings++
	} else {
		if err := printStatus("   ✅ Clipboard timeout: %v (secure)\n", cfg.ClipboardTTL); err != nil {
			return err
		}
	}

	// Check auto-lock timeout
	if cfg.AutoLockTTL > 4*time.Hour {
		if err := printStatus("   ⚠️  Auto-lock timeout is %v (consider reducing for better security)\n", cfg.AutoLockTTL); err != nil {
			return err
		}
		warnings++
	} else {
		if err := printStatus("   ✅ Auto-lock timeout: %v (secure)\n", cfg.AutoLockTTL); err != nil {
			return err
		}
	}

	// Summary
	if err := printStatus("\nSummary:\n"); err != nil {
		return err
	}

	if issues > 0 {
		if err := printStatus("   ❌ Found %d critical issues that need attention\n", issues); err != nil {
			return err
		}
	} else {
		if err := printStatus("   ✅ No critical issues found\n"); err != nil {
			return err
		}
	}

	if warnings > 0 {
		if err := printStatus("   ⚠️  Found %d warnings to review\n", warnings); err != nil {
			return err
		}
	} else {
		if err := printStatus("   ✅ No warnings found\n"); err != nil {
			return err
		}
	}

	switch {
	case issues == 0 && warnings == 0:
		if err := printStatus("\n✅ Your vault is in excellent condition!\n"); err != nil {
			return err
		}
	case issues == 0:
		if err := printStatus("\nℹ️  Your vault is secure, but there are some recommendations to consider.\n"); err != nil {
			return err
		}
	default:
		if err := printStatus("\n❌ Please address the critical issues listed above.\n"); err != nil {
			return err
		}
	}

	// Additional recommendations
	if err := printStatus("\nAdditional Security Recommendations:\n"); err != nil {
		return err
	}
	if err := printStatus("• Use a strong, unique master passphrase\n"); err != nil {
		return err
	}
	if err := printStatus("• Enable full-disk encryption on your system\n"); err != nil {
		return err
	}
	if err := printStatus("• Keep your vault software updated\n"); err != nil {
		return err
	}
	if err := printStatus("• Regularly backup your encrypted vault file\n"); err != nil {
		return err
	}
	if err := printStatus("• Use 'vault lock' when not actively using the vault\n"); err != nil {
		return err
	}

	return nil
}
