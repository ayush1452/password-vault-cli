package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// PromptPassword prompts for a password without echoing to terminal
func PromptPassword(prompt string) (string, error) {
	fmt.Print(prompt)

	// Get file descriptor for stdin
	fd := int(syscall.Stdin)

	// Read password without echo
	password, err := term.ReadPassword(fd)
	fmt.Println() // Print newline after password input

	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}

	return string(password), nil
}

// PromptPasswordConfirm prompts for a password and confirmation
func PromptPasswordConfirm(prompt string) (string, error) {
	password, err := PromptPassword(prompt)
	if err != nil {
		return "", err
	}

	confirm, err := PromptPassword("Confirm password: ")
	if err != nil {
		return "", err
	}

	if password != confirm {
		return "", fmt.Errorf("passwords do not match")
	}

	return password, nil
}

// PromptInput prompts for regular input
func PromptInput(prompt string) (string, error) {
	fmt.Print(prompt)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	return strings.TrimSpace(input), nil
}

// PromptConfirm prompts for yes/no confirmation
func PromptConfirm(prompt string, defaultYes bool) (bool, error) {
	var suffix string
	if defaultYes {
		suffix = " [Y/n]: "
	} else {
		suffix = " [y/N]: "
	}

	input, err := PromptInput(prompt + suffix)
	if err != nil {
		return false, err
	}

	input = strings.ToLower(strings.TrimSpace(input))

	if input == "" {
		return defaultYes, nil
	}

	return input == "y" || input == "yes", nil
}

// PromptChoice prompts for a choice from a list of options
func PromptChoice(prompt string, choices []string) (string, error) {
	fmt.Println(prompt)
	for i, choice := range choices {
		fmt.Printf("  %d) %s\n", i+1, choice)
	}

	input, err := PromptInput("Enter choice (1-" + fmt.Sprintf("%d", len(choices)) + "): ")
	if err != nil {
		return "", err
	}

	// Try to parse as number
	var choice int
	if _, err := fmt.Sscanf(input, "%d", &choice); err == nil {
		if choice >= 1 && choice <= len(choices) {
			return choices[choice-1], nil
		}
	}

	// Try to match as string
	input = strings.ToLower(input)
	for _, choice := range choices {
		if strings.ToLower(choice) == input {
			return choice, nil
		}
	}

	return "", fmt.Errorf("invalid choice: %s", input)
}
