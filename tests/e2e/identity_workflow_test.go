package e2e_test

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestIdentityWorkflow(t *testing.T) {
	helper := NewTestHelper(t)
	helper.InitVault()
	if err := helper.UnlockVault(); err != nil {
		t.Fatalf("failed to unlock vault: %v", err)
	}

	if stdout, stderr, err := helper.RunCommand("did", "create", "issuer"); err != nil {
		t.Fatalf("did create issuer failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	if stdout, stderr, err := helper.RunCommand("did", "create", "subject"); err != nil {
		t.Fatalf("did create subject failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	stdout, stderr, err := helper.RunCommand(
		"vc", "issue", "employee-cred",
		"--issuer", "issuer",
		"--subject", "subject",
		"--type", "EmployeeCredential",
		"--claim", "role=admin",
	)
	if err != nil {
		t.Fatalf("vc issue failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}

	proofPath := filepath.Join(helper.tempDir, "proof.json")
	stdout, stderr, err = helper.RunCommand(
		"zk-proof",
		"--did", "issuer",
		"--challenge", "login-challenge",
		"--output", proofPath,
	)
	if err != nil {
		t.Fatalf("zk-proof generate failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "\"response\"") {
		t.Fatalf("expected proof JSON output, got: %s", stdout)
	}

	stdout, stderr, err = helper.RunCommand(
		"zk-proof", "verify",
		"--did", "issuer",
		"--challenge", "login-challenge",
		"--proof", proofPath,
	)
	if err != nil {
		t.Fatalf("zk-proof verify failed: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	if !strings.Contains(stdout, "verified successfully") {
		t.Fatalf("expected verification success output, got stdout=%s stderr=%s", stdout, stderr)
	}
}
