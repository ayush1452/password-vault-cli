package identity

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// GenerateIdentity creates a new did:jwk identity using a P-256 keypair.
func GenerateIdentity(name string, now time.Time) (*IdentityRecord, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("identity name cannot be empty")
	}

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ECDSA key: %w", err)
	}

	publicJWK, privateJWK, err := jwkFromPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	did, err := buildDIDFromJWK(publicJWK)
	if err != nil {
		return nil, err
	}

	verificationMethodID := did + VerificationMethodSuffix
	document := DIDDocument{
		Context: []string{"https://www.w3.org/ns/did/v1"},
		ID:      did,
		VerificationMethod: []VerificationMethod{
			{
				ID:           verificationMethodID,
				Type:         VerificationMethodType,
				Controller:   did,
				PublicKeyJWK: publicJWK,
			},
		},
		Authentication:  []string{verificationMethodID},
		AssertionMethod: []string{verificationMethodID},
	}

	return &IdentityRecord{
		Name:                 name,
		DID:                  did,
		VerificationMethodID: verificationMethodID,
		PublicJWK:            publicJWK,
		PrivateJWK:           privateJWK,
		Document:             document,
		CreatedAt:            now.UTC(),
	}, nil
}

// PublicCopy removes the private key material from an identity record.
func (r *IdentityRecord) PublicCopy() *IdentityRecord {
	if r == nil {
		return nil
	}

	clone := *r
	clone.PrivateJWK = JWK{}
	return &clone
}

// PublicDocumentJSON encodes the public DID document for export and display.
func (r *IdentityRecord) PublicDocumentJSON() ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf("identity record is nil")
	}
	return CanonicalizeJSON(r.Document)
}

// ParseDIDInput extracts the public key information from a did:jwk string.
func ParseDIDInput(did string) (*DIDDocument, *ecdsa.PublicKey, error) {
	if !strings.HasPrefix(did, DIDMethodPrefix) {
		return nil, nil, fmt.Errorf("unsupported DID method: %s", did)
	}

	encoded := strings.TrimPrefix(did, DIDMethodPrefix)
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, nil, fmt.Errorf("decode did:jwk payload: %w", err)
	}

	var publicJWK JWK
	if err := jsonUnmarshal(raw, &publicJWK); err != nil {
		return nil, nil, fmt.Errorf("decode did:jwk JWK: %w", err)
	}

	publicKey, err := publicKeyFromJWK(publicJWK)
	if err != nil {
		return nil, nil, err
	}

	document := DIDDocument{
		Context: []string{"https://www.w3.org/ns/did/v1"},
		ID:      did,
		VerificationMethod: []VerificationMethod{
			{
				ID:           did + VerificationMethodSuffix,
				Type:         VerificationMethodType,
				Controller:   did,
				PublicKeyJWK: publicJWK,
			},
		},
		Authentication:  []string{did + VerificationMethodSuffix},
		AssertionMethod: []string{did + VerificationMethodSuffix},
	}

	return &document, publicKey, nil
}

func buildDIDFromJWK(publicJWK JWK) (string, error) {
	payload, err := CanonicalizeJSON(publicJWK)
	if err != nil {
		return "", fmt.Errorf("canonicalize public JWK: %w", err)
	}
	return DIDMethodPrefix + base64.RawURLEncoding.EncodeToString(payload), nil
}

func jwkFromPrivateKey(privateKey *ecdsa.PrivateKey) (JWK, JWK, error) {
	if privateKey == nil {
		return JWK{}, JWK{}, fmt.Errorf("private key is nil")
	}
	if privateKey.Curve != elliptic.P256() {
		return JWK{}, JWK{}, fmt.Errorf("unsupported curve")
	}

	x := base64.RawURLEncoding.EncodeToString(paddedScalar(privateKey.PublicKey.X, 32))
	y := base64.RawURLEncoding.EncodeToString(paddedScalar(privateKey.PublicKey.Y, 32))
	d := base64.RawURLEncoding.EncodeToString(paddedScalar(privateKey.D, 32))

	publicJWK := JWK{
		Kty: "EC",
		Crv: "P-256",
		X:   x,
		Y:   y,
		Alg: "ES256",
		Use: "sig",
	}

	privateJWK := publicJWK
	privateJWK.D = d

	return publicJWK, privateJWK, nil
}

func publicKeyFromJWK(jwk JWK) (*ecdsa.PublicKey, error) {
	if jwk.Kty != "EC" || jwk.Crv != "P-256" {
		return nil, fmt.Errorf("unsupported JWK key type %s/%s", jwk.Kty, jwk.Crv)
	}

	xBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
	if err != nil {
		return nil, fmt.Errorf("decode JWK x coordinate: %w", err)
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(jwk.Y)
	if err != nil {
		return nil, fmt.Errorf("decode JWK y coordinate: %w", err)
	}

	publicKey := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}

	if !publicKey.Curve.IsOnCurve(publicKey.X, publicKey.Y) {
		return nil, fmt.Errorf("JWK public key is not on P-256")
	}

	return publicKey, nil
}

func privateKeyFromJWK(jwk JWK) (*ecdsa.PrivateKey, error) {
	publicKey, err := publicKeyFromJWK(jwk)
	if err != nil {
		return nil, err
	}
	if jwk.D == "" {
		return nil, fmt.Errorf("JWK does not include private key material")
	}

	dBytes, err := base64.RawURLEncoding.DecodeString(jwk.D)
	if err != nil {
		return nil, fmt.Errorf("decode JWK private scalar: %w", err)
	}

	privateKey := &ecdsa.PrivateKey{
		PublicKey: *publicKey,
		D:         new(big.Int).SetBytes(dBytes),
	}

	return privateKey, nil
}

func paddedScalar(value *big.Int, size int) []byte {
	output := make([]byte, size)
	if value == nil {
		return output
	}
	valueBytes := value.Bytes()
	copy(output[len(output)-len(valueBytes):], valueBytes)
	return output
}
