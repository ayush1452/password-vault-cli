package identity

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestGenerateIdentity(t *testing.T) {
	record, err := GenerateIdentity("alice", time.Unix(1700000000, 0))
	if err != nil {
		t.Fatalf("GenerateIdentity returned error: %v", err)
	}

	if record.Name != "alice" {
		t.Fatalf("unexpected identity name: %s", record.Name)
	}
	if !strings.HasPrefix(record.DID, DIDMethodPrefix) {
		t.Fatalf("unexpected DID: %s", record.DID)
	}
	if record.PrivateJWK.D == "" {
		t.Fatalf("expected private key material to be populated")
	}

	document, publicKey, err := ParseDIDInput(record.DID)
	if err != nil {
		t.Fatalf("ParseDIDInput returned error: %v", err)
	}
	if document.ID != record.DID {
		t.Fatalf("document ID mismatch")
	}
	if publicKey.X.Sign() == 0 || publicKey.Y.Sign() == 0 {
		t.Fatalf("public key coordinates were not populated")
	}
}

func TestCanonicalizeJSONIsDeterministic(t *testing.T) {
	left, err := CanonicalizeJSON(map[string]interface{}{
		"z": 1,
		"a": map[string]interface{}{
			"b": "2",
			"a": "1",
		},
	})
	if err != nil {
		t.Fatalf("CanonicalizeJSON returned error: %v", err)
	}

	right, err := CanonicalizeJSON(map[string]interface{}{
		"a": map[string]interface{}{
			"a": "1",
			"b": "2",
		},
		"z": 1,
	})
	if err != nil {
		t.Fatalf("CanonicalizeJSON returned error: %v", err)
	}

	if string(left) != string(right) {
		t.Fatalf("expected canonical JSON to be deterministic\nleft:  %s\nright: %s", left, right)
	}
}

func TestIssueAndVerifyCredential(t *testing.T) {
	issuer, err := GenerateIdentity("issuer", time.Unix(1700000000, 0))
	if err != nil {
		t.Fatalf("GenerateIdentity returned error: %v", err)
	}

	record, err := IssueCredential(
		issuer,
		"cred-1",
		"did:jwk:test-subject",
		[]string{"EmployeeCredential"},
		[]CredentialClaim{
			{Name: "role", Value: "admin"},
			{Name: "team", Value: "platform"},
		},
		nil,
		time.Unix(1700000100, 0),
	)
	if err != nil {
		t.Fatalf("IssueCredential returned error: %v", err)
	}

	if err := VerifyCredential(record); err != nil {
		t.Fatalf("VerifyCredential returned error: %v", err)
	}

	payload, err := MarshalCredentialJSON(record)
	if err != nil {
		t.Fatalf("MarshalCredentialJSON returned error: %v", err)
	}

	var document map[string]interface{}
	if err := json.Unmarshal(payload, &document); err != nil {
		t.Fatalf("credential output is not valid JSON: %v", err)
	}
	if document["issuer"] != issuer.DID {
		t.Fatalf("issuer mismatch in marshaled VC")
	}
}

func TestVerifyCredentialRejectsTampering(t *testing.T) {
	issuer, err := GenerateIdentity("issuer", time.Unix(1700000000, 0))
	if err != nil {
		t.Fatalf("GenerateIdentity returned error: %v", err)
	}

	record, err := IssueCredential(
		issuer,
		"cred-2",
		"did:jwk:test-subject",
		[]string{"AccessCredential"},
		[]CredentialClaim{{Name: "scope", Value: "read"}},
		nil,
		time.Unix(1700000100, 0),
	)
	if err != nil {
		t.Fatalf("IssueCredential returned error: %v", err)
	}

	record.Claims[0].Value = "write"
	if err := VerifyCredential(record); err == nil {
		t.Fatalf("expected tampered credential verification to fail")
	}
}

func TestGenerateAndVerifyKeyPossessionProof(t *testing.T) {
	identityRecord, err := GenerateIdentity("alice", time.Unix(1700000000, 0))
	if err != nil {
		t.Fatalf("GenerateIdentity returned error: %v", err)
	}

	proof, err := GenerateKeyPossessionProof(identityRecord, "auth-challenge", time.Unix(1700000200, 0))
	if err != nil {
		t.Fatalf("GenerateKeyPossessionProof returned error: %v", err)
	}

	if err := VerifyKeyPossessionProof(identityRecord.DID, proof, "auth-challenge"); err != nil {
		t.Fatalf("VerifyKeyPossessionProof returned error: %v", err)
	}
}

func TestKeyPossessionProofRejectsWrongChallenge(t *testing.T) {
	identityRecord, err := GenerateIdentity("alice", time.Unix(1700000000, 0))
	if err != nil {
		t.Fatalf("GenerateIdentity returned error: %v", err)
	}

	proof, err := GenerateKeyPossessionProof(identityRecord, "auth-challenge", time.Unix(1700000200, 0))
	if err != nil {
		t.Fatalf("GenerateKeyPossessionProof returned error: %v", err)
	}

	if err := VerifyKeyPossessionProof(identityRecord.DID, proof, "other-challenge"); err == nil {
		t.Fatalf("expected verification to fail for a wrong challenge")
	}
}

func TestKeyPossessionProofRejectsWrongDID(t *testing.T) {
	identityRecord, err := GenerateIdentity("alice", time.Unix(1700000000, 0))
	if err != nil {
		t.Fatalf("GenerateIdentity returned error: %v", err)
	}
	otherRecord, err := GenerateIdentity("bob", time.Unix(1700000001, 0))
	if err != nil {
		t.Fatalf("GenerateIdentity returned error: %v", err)
	}

	proof, err := GenerateKeyPossessionProof(identityRecord, "auth-challenge", time.Unix(1700000200, 0))
	if err != nil {
		t.Fatalf("GenerateKeyPossessionProof returned error: %v", err)
	}

	if err := VerifyKeyPossessionProof(otherRecord.DID, proof, "auth-challenge"); err == nil {
		t.Fatalf("expected verification to fail for the wrong DID")
	}
}

func TestIdentityRecordSerializationRoundTrip(t *testing.T) {
	record, err := GenerateIdentity("alice", time.Unix(1700000000, 0))
	if err != nil {
		t.Fatalf("GenerateIdentity returned error: %v", err)
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}

	var decoded IdentityRecord
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}

	if decoded.DID != record.DID {
		t.Fatalf("decoded DID mismatch")
	}
	if decoded.PrivateJWK.D == "" {
		t.Fatalf("expected private key material after round-trip")
	}
}
