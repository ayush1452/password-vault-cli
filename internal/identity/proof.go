package identity

import (
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"
)

// GenerateKeyPossessionProof creates a Schnorr proof of DID key possession for a challenge.
func GenerateKeyPossessionProof(record *IdentityRecord, challenge string, now time.Time) (*KeyPossessionProof, error) {
	if record == nil {
		return nil, fmt.Errorf("identity record is nil")
	}
	if challenge == "" {
		return nil, fmt.Errorf("challenge cannot be empty")
	}

	privateKey, err := privateKeyFromJWK(record.PrivateJWK)
	if err != nil {
		return nil, err
	}

	curve := elliptic.P256()
	order := curve.Params().N
	k, err := rand.Int(rand.Reader, order)
	if err != nil {
		return nil, fmt.Errorf("generate proof nonce: %w", err)
	}
	if k.Sign() == 0 {
		k = big.NewInt(1)
	}

	commitmentX, commitmentY := curve.ScalarBaseMult(paddedScalar(k, 32))
	challengeScalar := deriveProofChallenge(record.DID, challenge, commitmentX, commitmentY)

	response := new(big.Int).Mul(challengeScalar, privateKey.D)
	response.Add(response, k)
	response.Mod(response, order)

	challengeDigest := sha256.Sum256([]byte(challenge))

	return &KeyPossessionProof{
		Type:         KeyProofType,
		DID:          record.DID,
		CommitmentX:  base64.RawURLEncoding.EncodeToString(paddedScalar(commitmentX, 32)),
		CommitmentY:  base64.RawURLEncoding.EncodeToString(paddedScalar(commitmentY, 32)),
		Response:     base64.RawURLEncoding.EncodeToString(paddedScalar(response, 32)),
		ChallengeSHA: hex.EncodeToString(challengeDigest[:]),
		GeneratedAt:  now.UTC(),
	}, nil
}

// VerifyKeyPossessionProof verifies a proof against a did:jwk input and challenge.
func VerifyKeyPossessionProof(did string, proof *KeyPossessionProof, challenge string) error {
	if proof == nil {
		return fmt.Errorf("proof is nil")
	}
	if proof.Type != KeyProofType {
		return fmt.Errorf("unsupported proof type: %s", proof.Type)
	}
	if did == "" {
		return fmt.Errorf("DID cannot be empty")
	}
	if challenge == "" {
		return fmt.Errorf("challenge cannot be empty")
	}
	if proof.DID != did {
		return fmt.Errorf("proof DID does not match requested DID")
	}

	_, publicKey, err := ParseDIDInput(did)
	if err != nil {
		return err
	}

	challengeDigest := sha256.Sum256([]byte(challenge))
	if proof.ChallengeSHA != hex.EncodeToString(challengeDigest[:]) {
		return fmt.Errorf("challenge digest does not match proof")
	}

	commitmentXBytes, err := base64.RawURLEncoding.DecodeString(proof.CommitmentX)
	if err != nil {
		return fmt.Errorf("decode proof commitment_x: %w", err)
	}
	commitmentYBytes, err := base64.RawURLEncoding.DecodeString(proof.CommitmentY)
	if err != nil {
		return fmt.Errorf("decode proof commitment_y: %w", err)
	}
	responseBytes, err := base64.RawURLEncoding.DecodeString(proof.Response)
	if err != nil {
		return fmt.Errorf("decode proof response: %w", err)
	}

	curve := elliptic.P256()
	order := curve.Params().N
	commitmentX := new(big.Int).SetBytes(commitmentXBytes)
	commitmentY := new(big.Int).SetBytes(commitmentYBytes)
	response := new(big.Int).SetBytes(responseBytes)
	if response.Sign() == 0 || response.Cmp(order) >= 0 {
		return fmt.Errorf("proof response is out of range")
	}
	if !curve.IsOnCurve(commitmentX, commitmentY) {
		return fmt.Errorf("proof commitment is not on P-256")
	}

	challengeScalar := deriveProofChallenge(did, challenge, commitmentX, commitmentY)
	lhsX, lhsY := curve.ScalarBaseMult(paddedScalar(response, 32))
	rhsPubX, rhsPubY := curve.ScalarMult(publicKey.X, publicKey.Y, paddedScalar(challengeScalar, 32))
	rhsX, rhsY := curve.Add(commitmentX, commitmentY, rhsPubX, rhsPubY)

	if lhsX.Cmp(rhsX) != 0 || lhsY.Cmp(rhsY) != 0 {
		return fmt.Errorf("proof verification failed")
	}

	return nil
}

func deriveProofChallenge(did, challenge string, commitmentX, commitmentY *big.Int) *big.Int {
	input := []byte(did + "|" + challenge + "|" +
		base64.RawURLEncoding.EncodeToString(paddedScalar(commitmentX, 32)) + "|" +
		base64.RawURLEncoding.EncodeToString(paddedScalar(commitmentY, 32)))
	digest := sha256.Sum256(input)
	scalar := new(big.Int).SetBytes(digest[:])
	scalar.Mod(scalar, elliptic.P256().Params().N)
	if scalar.Sign() == 0 {
		scalar = big.NewInt(1)
	}
	return scalar
}
