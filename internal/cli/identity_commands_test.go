package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDIDCreateAndShowCommand(t *testing.T) {
	helper := NewTestHelper(t)
	helper.SetupVault(t, "test-passphrase")

	cleanup := helper.unlockWithSession(t)
	defer cleanup()

	createCmd := NewDID(helper.Config)
	var createOut, createErr bytes.Buffer
	createCmd.SetOut(&createOut)
	createCmd.SetErr(&createErr)
	createCmd.SetArgs([]string{"create", "issuer", "--json"})

	if err := createCmd.Execute(); err != nil {
		t.Fatalf("did create failed: %v", err)
	}
	if !strings.Contains(createOut.String(), "did:jwk:") {
		t.Fatalf("expected did:jwk output, got: %s", createOut.String())
	}

	showCmd := NewDID(helper.Config)
	var showOut, showErr bytes.Buffer
	showCmd.SetOut(&showOut)
	showCmd.SetErr(&showErr)
	showCmd.SetArgs([]string{"show", "issuer"})

	if err := showCmd.Execute(); err != nil {
		t.Fatalf("did show failed: %v", err)
	}
	if !strings.Contains(showOut.String(), "Verification Method:") {
		t.Fatalf("expected DID show output, got: %s", showOut.String())
	}
}

func TestVCIssueAndVerifyCommand(t *testing.T) {
	helper := NewTestHelper(t)
	helper.SetupVault(t, "test-passphrase")

	cleanup := helper.unlockWithSession(t)
	defer cleanup()

	for _, name := range []string{"issuer", "subject"} {
		cmd := NewDID(helper.Config)
		cmd.SetOut(&bytes.Buffer{})
		cmd.SetErr(&bytes.Buffer{})
		cmd.SetArgs([]string{"create", name})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("did create %s failed: %v", name, err)
		}
	}

	issueCmd := NewVC(helper.Config)
	var issueOut, issueErr bytes.Buffer
	issueCmd.SetOut(&issueOut)
	issueCmd.SetErr(&issueErr)
	issueCmd.SetArgs([]string{
		"issue", "employee-cred",
		"--issuer", "issuer",
		"--subject", "subject",
		"--type", "EmployeeCredential",
		"--claim", "role=admin",
		"--json",
	})
	if err := issueCmd.Execute(); err != nil {
		t.Fatalf("vc issue failed: %v", err)
	}
	if !strings.Contains(issueOut.String(), "\"proof\"") {
		t.Fatalf("expected signed VC JSON, got: %s", issueOut.String())
	}

	verifyCmd := NewVC(helper.Config)
	var verifyOut, verifyErr bytes.Buffer
	verifyCmd.SetOut(&verifyOut)
	verifyCmd.SetErr(&verifyErr)
	verifyCmd.SetArgs([]string{"verify", "employee-cred"})
	if err := verifyCmd.Execute(); err != nil {
		t.Fatalf("vc verify failed: %v", err)
	}
	if !strings.Contains(verifyOut.String(), "verified successfully") {
		t.Fatalf("expected verify success output, got: %s", verifyOut.String())
	}
}

func TestZKProofGenerateAndVerifyCommand(t *testing.T) {
	helper := NewTestHelper(t)
	helper.SetupVault(t, "test-passphrase")

	cleanup := helper.unlockWithSession(t)
	defer cleanup()

	didCmd := NewDID(helper.Config)
	didCmd.SetOut(&bytes.Buffer{})
	didCmd.SetErr(&bytes.Buffer{})
	didCmd.SetArgs([]string{"create", "issuer"})
	if err := didCmd.Execute(); err != nil {
		t.Fatalf("did create failed: %v", err)
	}

	proofPath := filepath.Join(helper.TempDir, "proof.json")
	generateCmd := NewZKProof(helper.Config)
	var generateOut, generateErr bytes.Buffer
	generateCmd.SetOut(&generateOut)
	generateCmd.SetErr(&generateErr)
	generateCmd.SetArgs([]string{
		"--did", "issuer",
		"--challenge", "login-challenge",
		"--output", proofPath,
	})
	if err := generateCmd.Execute(); err != nil {
		t.Fatalf("zk-proof generate failed: %v", err)
	}
	if !strings.Contains(generateOut.String(), "\"response\"") {
		t.Fatalf("expected proof JSON output, got: %s", generateOut.String())
	}
	if _, err := os.Stat(proofPath); err != nil {
		t.Fatalf("expected proof file to exist: %v", err)
	}

	verifyCmd := NewZKProof(helper.Config)
	var verifyOut, verifyErr bytes.Buffer
	verifyCmd.SetOut(&verifyOut)
	verifyCmd.SetErr(&verifyErr)
	verifyCmd.SetArgs([]string{
		"verify",
		"--did", "issuer",
		"--challenge", "login-challenge",
		"--proof", proofPath,
	})
	if err := verifyCmd.Execute(); err != nil {
		t.Fatalf("zk-proof verify failed: %v", err)
	}
	if !strings.Contains(verifyOut.String(), "verified successfully") {
		t.Fatalf("expected verification success output, got: %s", verifyOut.String())
	}
}
