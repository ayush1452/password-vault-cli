package main

import (
	"fmt"
)

var vault = make(map[string]string)

// addPassword stores a password for a given service
func addPassword(service string, password string) {
	vault[service] = password
	fmt.Println("Password added successfully. ")
}

// getPassword retrieves a password for a given service
func getPassword(service string) {
	password, exists := vault[service]
	if !exists {
		fmt.Println("No password is found for service:", service)
		return
	}
	fmt.Println("Password for", service, "is:", password)
}

// listServices prints all stored services
func listServices() {
	if len(vault) == 0 {
		fmt.Println("No services stored.")
		return
	}

	fmt.Println("Stored services:")
	for service := range vault {
		fmt.Println("-", service)
	}
}

func main() {
	// Example usage
	addPassword("gmail", "gmail_password_123")
	addPassword("github", "github_password_456")

	listServices()
	getPassword("gmail")
	getPassword("facebook")
}

// TestVaultOperations tests the basic vault functionality
func TestVaultOperations() {
	// Clear vault for clean test
	vault = make(map[string]string)

	// Test adding passwords
	addPassword("test-service", "test-password-123")
	addPassword("another-service", "another-password-456")

	// Test listing services
	fmt.Println("\n--- Testing listServices ---")
	listServices()

	// Test retrieving existing password
	fmt.Println("\n--- Testing getPassword (existing) ---")
	getPassword("test-service")

	// Test retrieving non-existing password
	fmt.Println("\n--- Testing getPassword (non-existing) ---")
	getPassword("nonexistent-service")

	// Test empty vault
	fmt.Println("\n--- Testing empty vault ---")
	vault = make(map[string]string)
	listServices()

	fmt.Println("\n--- Test completed ---")
}
