package identity

import "time"

const (
	DIDMethodPrefix           = "did:jwk:"
	VerificationMethodSuffix  = "#0"
	VerificationMethodType    = "JsonWebKey2020"
	VerifiableCredentialType  = "VerifiableCredential"
	VerifiableCredentialCtx   = "https://www.w3.org/2018/credentials/v1"
	VerifiableCredentialProof = "JsonWebSignature2020"
	KeyProofType              = "SchnorrP256"
)

// JWK represents an EC JSON Web Key used by did:jwk.
type JWK struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
	Alg string `json:"alg,omitempty"`
	Use string `json:"use,omitempty"`
	Kid string `json:"kid,omitempty"`
	D   string `json:"d,omitempty"`
}

// VerificationMethod describes a DID verification method.
type VerificationMethod struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Controller   string `json:"controller"`
	PublicKeyJWK JWK    `json:"publicKeyJwk"`
}

// DIDDocument is the public document stored for an identity.
type DIDDocument struct {
	Context            []string             `json:"@context,omitempty"`
	ID                 string               `json:"id"`
	VerificationMethod []VerificationMethod `json:"verificationMethod"`
	Authentication     []string             `json:"authentication,omitempty"`
	AssertionMethod    []string             `json:"assertionMethod,omitempty"`
}

// IdentityRecord stores a local DID and its encrypted private key material.
type IdentityRecord struct {
	Name                 string      `json:"name"`
	DID                  string      `json:"did"`
	VerificationMethodID string      `json:"verification_method_id"`
	PublicJWK            JWK         `json:"public_jwk"`
	PrivateJWK           JWK         `json:"private_jwk,omitempty"`
	Document             DIDDocument `json:"document"`
	CreatedAt            time.Time   `json:"created_at"`
}

// CredentialClaim is a flat key=value claim used by the MVP VC implementation.
type CredentialClaim struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// CredentialProof stores ES256 signature material for a VC.
type CredentialProof struct {
	Type               string    `json:"type"`
	Created            time.Time `json:"created"`
	ProofPurpose       string    `json:"proof_purpose"`
	VerificationMethod string    `json:"verification_method"`
	Cryptosuite        string    `json:"cryptosuite"`
	JWS                string    `json:"jws"`
}

// CredentialRecord is the stored verifiable credential representation.
type CredentialRecord struct {
	ID         string            `json:"id"`
	IssuerName string            `json:"issuer_name,omitempty"`
	IssuerDID  string            `json:"issuer_did"`
	Subject    string            `json:"subject"`
	Types      []string          `json:"types"`
	Claims     []CredentialClaim `json:"claims"`
	IssuedAt   time.Time         `json:"issued_at"`
	ExpiresAt  *time.Time        `json:"expires_at,omitempty"`
	Proof      CredentialProof   `json:"proof"`
}

// KeyPossessionProof stores a non-interactive Schnorr proof over P-256.
type KeyPossessionProof struct {
	Type         string    `json:"type"`
	DID          string    `json:"did"`
	CommitmentX  string    `json:"commitment_x"`
	CommitmentY  string    `json:"commitment_y"`
	Response     string    `json:"response"`
	ChallengeSHA string    `json:"challenge_sha256"`
	GeneratedAt  time.Time `json:"generated_at"`
}
