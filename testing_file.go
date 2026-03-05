package main

import (
	"fmt"
)

var vault = make(map[string]string)

// addPassword stores a password for a given service
func addPassword(service string, password string) {
	vault[service] = password
	fmt.Println("Password added successfully.")
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
