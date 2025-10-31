package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	fmt.Println("Vault Security & Health Check")
	fmt.Println("=============================")

	issues := 0
	warnings := 0

	// Check 1: Vault file existence and permissions
	fmt.Println("\n1. Vault File Security")
	if _, err := os.Stat(vaultPath); os.IsNotExist(err) {
		fmt.Printf("   ❌ Vault file not found: %s\n", vaultPath)
		issues++
	} else {
		info, err := os.Stat(vaultPath)
		if err != nil {
			fmt.Printf("   ❌ Cannot check vault file: %v\n", err)
			issues++
		} else {
			perm := info.Mode().Perm()
			if perm == 0600 {
				fmt.Printf("   ✅ Vault file permissions: %o (secure)\n", perm)
			} else if perm&0077 != 0 {
				fmt.Printf("   ❌ Vault file permissions: %o (too permissive, should be 0600)\n", perm)
				fmt.Printf("      Fix with: chmod 600 %s\n", vaultPath)
				issues++
			} else {
				fmt.Printf("   ⚠️  Vault file permissions: %o (acceptable but 0600 recommended)\n", perm)
				warnings++
			}
		}
	}

	// Check 2: Configuration file security
	fmt.Println("\n2. Configuration Security")
	if cfgFile != "" {
		if info, err := os.Stat(cfgFile); err == nil {
			perm := info.Mode().Perm()
			if perm == 0600 {
				fmt.Printf("   ✅ Config file permissions: %o (secure)\n", perm)
			} else if perm&0077 != 0 {
				fmt.Printf("   ❌ Config file permissions: %o (too permissive, should be 0600)\n", perm)
				fmt.Printf("      Fix with: chmod 600 %s\n", cfgFile)
				issues++
			} else {
				fmt.Printf("   ⚠️  Config file permissions: %o (acceptable but 0600 recommended)\n", perm)
				warnings++
			}
		} else {
			fmt.Printf("   ✅ Config file not found (using defaults)\n")
		}
	}

	// Check 3: Directory permissions
	fmt.Println("\n3. Directory Security")
	vaultDir := filepath.Dir(vaultPath)
	if info, err := os.Stat(vaultDir); err == nil {
		perm := info.Mode().Perm()
		if perm&0077 == 0 {
			fmt.Printf("   ✅ Vault directory permissions: %o (secure)\n", perm)
		} else {
			fmt.Printf("   ⚠️  Vault directory permissions: %o (consider 0700 for better security)\n", perm)
			warnings++
		}
	}

	// Check 4: Vault integrity (if unlocked)
	fmt.Println("\n4. Vault Integrity")
	if IsUnlocked() {
		vaultStore := GetVaultStore()
		if err := vaultStore.VerifyIntegrity(); err != nil {
			fmt.Printf("   ❌ Vault integrity check failed: %v\n", err)
			issues++
		} else {
			fmt.Printf("   ✅ Vault structure is valid\n")
		}

		// Check metadata
		metadata, err := vaultStore.GetVaultMetadata()
		if err != nil {
			fmt.Printf("   ❌ Cannot read vault metadata: %v\n", err)
			issues++
		} else {
			fmt.Printf("   ✅ Vault version: %s\n", metadata.Version)
			fmt.Printf("   ✅ Created: %s\n", metadata.CreatedAt.Format("2006-01-02 15:04:05"))
		}
	} else {
		fmt.Printf("   ⚠️  Vault is locked, cannot perform integrity checks\n")
		fmt.Printf("      Run 'vault unlock' first for complete check\n")
		warnings++
	}

	// Check 5: KDF parameters (if available)
	fmt.Println("\n5. Cryptographic Parameters")
	if IsUnlocked() {
		vaultStore := GetVaultStore()
		metadata, err := vaultStore.GetVaultMetadata()
		if err == nil && metadata.KDFParams != nil {
			if memory, ok := metadata.KDFParams["memory"].(uint32); ok {
				if memory >= 65536 { // 64 MB
					fmt.Printf("   ✅ KDF memory parameter: %d KB (strong)\n", memory)
				} else if memory >= 8192 { // 8 MB
					fmt.Printf("   ⚠️  KDF memory parameter: %d KB (acceptable but consider increasing)\n", memory)
					warnings++
				} else {
					fmt.Printf("   ❌ KDF memory parameter: %d KB (weak, should be at least 8192 KB)\n", memory)
					issues++
				}
			}

			if iterations, ok := metadata.KDFParams["iterations"].(uint32); ok {
				if iterations >= 3 {
					fmt.Printf("   ✅ KDF iterations: %d (adequate)\n", iterations)
				} else {
					fmt.Printf("   ⚠️  KDF iterations: %d (consider increasing for better security)\n", iterations)
					warnings++
				}
			}

			if parallelism, ok := metadata.KDFParams["parallelism"].(uint8); ok {
				fmt.Printf("   ✅ KDF parallelism: %d\n", parallelism)
			}
		}
	}

	// Check 6: System security recommendations
	fmt.Println("\n6. System Security Recommendations")

	// Check for swap files
	if _, err := os.Stat("/proc/swaps"); err == nil {
		fmt.Printf("   ⚠️  Swap files detected - secrets may be written to disk\n")
		fmt.Printf("      Consider disabling swap or using encrypted swap\n")
		warnings++
	}

	// Check clipboard timeout
	if cfg.ClipboardTTL > 60*time.Second {
		fmt.Printf("   ⚠️  Clipboard timeout is %v (consider reducing for better security)\n", cfg.ClipboardTTL)
		warnings++
	} else {
		fmt.Printf("   ✅ Clipboard timeout: %v (secure)\n", cfg.ClipboardTTL)
	}

	// Check auto-lock timeout
	if cfg.AutoLockTTL > 4*time.Hour {
		fmt.Printf("   ⚠️  Auto-lock timeout is %v (consider reducing for better security)\n", cfg.AutoLockTTL)
		warnings++
	} else {
		fmt.Printf("   ✅ Auto-lock timeout: %v (secure)\n", cfg.AutoLockTTL)
	}

	// Summary
	fmt.Println("\n" + strings.Repeat("=", 40))
	if issues == 0 && warnings == 0 {
		fmt.Printf("✅ All checks passed! Your vault is secure.\n")
	} else {
		if issues > 0 {
			fmt.Printf("❌ Found %d security issues that should be fixed\n", issues)
		}
		if warnings > 0 {
			fmt.Printf("⚠️  Found %d warnings for consideration\n", warnings)
		}
	}

	// Additional recommendations
	fmt.Println("\nAdditional Security Recommendations:")
	fmt.Println("• Use a strong, unique master passphrase")
	fmt.Println("• Enable full-disk encryption on your system")
	fmt.Println("• Keep your vault software updated")
	fmt.Println("• Regularly backup your encrypted vault file")
	fmt.Println("• Use 'vault lock' when not actively using the vault")

	return nil
}
