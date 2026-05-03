package identity

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// NormalizeClaims trims, validates, sorts, and deduplicates flat credential claims.
func NormalizeClaims(claims []CredentialClaim) ([]CredentialClaim, error) {
	if len(claims) == 0 {
		return nil, fmt.Errorf("at least one claim is required")
	}

	normalized := make([]CredentialClaim, 0, len(claims))
	seen := make(map[string]struct{}, len(claims))
	for _, claim := range claims {
		name := strings.TrimSpace(claim.Name)
		if name == "" {
			return nil, fmt.Errorf("claim name cannot be empty")
		}
		if _, exists := seen[name]; exists {
			return nil, fmt.Errorf("duplicate claim name: %s", name)
		}
		seen[name] = struct{}{}
		normalized = append(normalized, CredentialClaim{
			Name:  name,
			Value: strings.TrimSpace(claim.Value),
		})
	}

	sort.Slice(normalized, func(i, j int) bool {
		return normalized[i].Name < normalized[j].Name
	})

	return normalized, nil
}

// NormalizeTypes prepares a stable, deduplicated VC type list.
func NormalizeTypes(types []string) ([]string, error) {
	seen := map[string]struct{}{
		VerifiableCredentialType: {},
	}
	normalized := []string{VerifiableCredentialType}
	for _, raw := range types {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}

	sort.Strings(normalized[1:])
	return normalized, nil
}

// IssueCredential creates and signs a verifiable credential from a local issuer DID.
func IssueCredential(
	issuer *IdentityRecord,
	credentialID, subject string,
	types []string,
	claims []CredentialClaim,
	expiresAt *time.Time,
	now time.Time,
) (*CredentialRecord, error) {
	if issuer == nil {
		return nil, fmt.Errorf("issuer identity is required")
	}
	if strings.TrimSpace(credentialID) == "" {
		return nil, fmt.Errorf("credential ID cannot be empty")
	}
	if strings.TrimSpace(subject) == "" {
		return nil, fmt.Errorf("credential subject cannot be empty")
	}

	normalizedClaims, err := NormalizeClaims(claims)
	if err != nil {
		return nil, err
	}
	normalizedTypes, err := NormalizeTypes(types)
	if err != nil {
		return nil, err
	}

	unsignedPayload, err := buildUnsignedCredentialPayload(
		strings.TrimSpace(credentialID),
		issuer.DID,
		strings.TrimSpace(subject),
		normalizedTypes,
		normalizedClaims,
		expiresAt,
		now.UTC(),
	)
	if err != nil {
		return nil, err
	}

	privateKey, err := privateKeyFromJWK(issuer.PrivateJWK)
	if err != nil {
		return nil, err
	}

	canonical, err := CanonicalizeJSON(unsignedPayload)
	if err != nil {
		return nil, fmt.Errorf("canonicalize unsigned credential: %w", err)
	}
	digest := sha256.Sum256(canonical)

	signature, err := ecdsa.SignASN1(rand.Reader, privateKey, digest[:])
	if err != nil {
		return nil, fmt.Errorf("sign verifiable credential: %w", err)
	}

	record := &CredentialRecord{
		ID:         strings.TrimSpace(credentialID),
		IssuerName: issuer.Name,
		IssuerDID:  issuer.DID,
		Subject:    strings.TrimSpace(subject),
		Types:      normalizedTypes,
		Claims:     normalizedClaims,
		IssuedAt:   now.UTC(),
		ExpiresAt:  normalizeOptionalTime(expiresAt),
		Proof: CredentialProof{
			Type:               VerifiableCredentialProof,
			Created:            now.UTC(),
			ProofPurpose:       "assertionMethod",
			VerificationMethod: issuer.VerificationMethodID,
			Cryptosuite:        "ES256",
			JWS:                base64.RawURLEncoding.EncodeToString(signature),
		},
	}

	return record, nil
}

// VerifyCredential verifies the signed VC proof against the issuer did:jwk.
func VerifyCredential(record *CredentialRecord) error {
	if record == nil {
		return fmt.Errorf("credential record is nil")
	}
	if strings.TrimSpace(record.IssuerDID) == "" {
		return fmt.Errorf("credential issuer DID is missing")
	}

	_, publicKey, err := ParseDIDInput(record.IssuerDID)
	if err != nil {
		return err
	}

	unsignedPayload, err := buildUnsignedCredentialPayload(
		record.ID,
		record.IssuerDID,
		record.Subject,
		record.Types,
		record.Claims,
		record.ExpiresAt,
		record.IssuedAt.UTC(),
	)
	if err != nil {
		return err
	}

	canonical, err := CanonicalizeJSON(unsignedPayload)
	if err != nil {
		return fmt.Errorf("canonicalize credential for verification: %w", err)
	}
	digest := sha256.Sum256(canonical)

	signature, err := base64.RawURLEncoding.DecodeString(record.Proof.JWS)
	if err != nil {
		return fmt.Errorf("decode credential signature: %w", err)
	}
	if !ecdsa.VerifyASN1(publicKey, digest[:], signature) {
		return fmt.Errorf("credential signature verification failed")
	}
	if record.ExpiresAt != nil && record.ExpiresAt.UTC().Before(time.Now().UTC()) {
		return fmt.Errorf("credential is expired")
	}
	return nil
}

// MarshalCredentialJSON renders the W3C-style VC document.
func MarshalCredentialJSON(record *CredentialRecord) ([]byte, error) {
	if record == nil {
		return nil, fmt.Errorf("credential record is nil")
	}

	document, err := buildSignedCredentialPayload(record)
	if err != nil {
		return nil, err
	}
	return CanonicalizeJSON(document)
}

// ParseCredentialJSON converts an exported VC JSON document back into a record.
func ParseCredentialJSON(data []byte) (*CredentialRecord, error) {
	var raw struct {
		ID                string                 `json:"id"`
		Type              []string               `json:"type"`
		Issuer            string                 `json:"issuer"`
		IssuanceDate      string                 `json:"issuanceDate"`
		ExpirationDate    string                 `json:"expirationDate"`
		CredentialSubject map[string]interface{} `json:"credentialSubject"`
		Proof             struct {
			Type               string `json:"type"`
			Created            string `json:"created"`
			ProofPurpose       string `json:"proofPurpose"`
			VerificationMethod string `json:"verificationMethod"`
			Cryptosuite        string `json:"cryptosuite"`
			JWS                string `json:"jws"`
		} `json:"proof"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal credential JSON: %w", err)
	}

	issuedAt, err := time.Parse(time.RFC3339, raw.IssuanceDate)
	if err != nil {
		return nil, fmt.Errorf("parse issuance date: %w", err)
	}

	var expiresAt *time.Time
	if strings.TrimSpace(raw.ExpirationDate) != "" {
		parsed, err := time.Parse(time.RFC3339, raw.ExpirationDate)
		if err != nil {
			return nil, fmt.Errorf("parse expiration date: %w", err)
		}
		expiresAt = &parsed
	}

	proofCreated, err := time.Parse(time.RFC3339, raw.Proof.Created)
	if err != nil {
		return nil, fmt.Errorf("parse proof created time: %w", err)
	}

	subject := ""
	claims := make([]CredentialClaim, 0, len(raw.CredentialSubject))
	for key, value := range raw.CredentialSubject {
		if key == "id" {
			subject, _ = value.(string)
			continue
		}
		stringValue, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("claim %s must be a string", key)
		}
		claims = append(claims, CredentialClaim{Name: key, Value: stringValue})
	}

	normalizedClaims, err := NormalizeClaims(claims)
	if err != nil {
		return nil, err
	}

	return &CredentialRecord{
		ID:        raw.ID,
		IssuerDID: raw.Issuer,
		Subject:   subject,
		Types:     raw.Type,
		Claims:    normalizedClaims,
		IssuedAt:  issuedAt.UTC(),
		ExpiresAt: expiresAt,
		Proof: CredentialProof{
			Type:               raw.Proof.Type,
			Created:            proofCreated.UTC(),
			ProofPurpose:       raw.Proof.ProofPurpose,
			VerificationMethod: raw.Proof.VerificationMethod,
			Cryptosuite:        raw.Proof.Cryptosuite,
			JWS:                raw.Proof.JWS,
		},
	}, nil
}

func buildUnsignedCredentialPayload(
	credentialID, issuerDID, subject string,
	types []string,
	claims []CredentialClaim,
	expiresAt *time.Time,
	issuedAt time.Time,
) (map[string]interface{}, error) {
	normalizedClaims, err := NormalizeClaims(claims)
	if err != nil {
		return nil, err
	}
	normalizedTypes, err := NormalizeTypes(types)
	if err != nil {
		return nil, err
	}

	subjectPayload := make(map[string]interface{}, len(normalizedClaims)+1)
	subjectPayload["id"] = subject
	for _, claim := range normalizedClaims {
		subjectPayload[claim.Name] = claim.Value
	}

	payload := map[string]interface{}{
		"@context":          []string{VerifiableCredentialCtx},
		"id":                credentialID,
		"type":              normalizedTypes,
		"issuer":            issuerDID,
		"issuanceDate":      issuedAt.UTC().Format(time.RFC3339),
		"credentialSubject": subjectPayload,
	}
	if expiresAt != nil {
		payload["expirationDate"] = expiresAt.UTC().Format(time.RFC3339)
	}

	return payload, nil
}

func buildSignedCredentialPayload(record *CredentialRecord) (map[string]interface{}, error) {
	payload, err := buildUnsignedCredentialPayload(
		record.ID,
		record.IssuerDID,
		record.Subject,
		record.Types,
		record.Claims,
		record.ExpiresAt,
		record.IssuedAt.UTC(),
	)
	if err != nil {
		return nil, err
	}

	payload["proof"] = map[string]interface{}{
		"type":               record.Proof.Type,
		"created":            record.Proof.Created.UTC().Format(time.RFC3339),
		"proofPurpose":       record.Proof.ProofPurpose,
		"verificationMethod": record.Proof.VerificationMethod,
		"cryptosuite":        record.Proof.Cryptosuite,
		"jws":                record.Proof.JWS,
	}

	return payload, nil
}

func normalizeOptionalTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	normalized := value.UTC()
	return &normalized
}
