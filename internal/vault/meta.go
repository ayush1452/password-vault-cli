package vault

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/vault-cli/vault/internal/domain"
)

// MetadataInfo captures cryptographic configuration stored in the vault header.
type MetadataInfo struct {
	Cipher          string       `json:"cipher"`
	KDF             Argon2Params `json:"kdf"`
	SaltLength      int          `json:"salt_length"`
	MetadataCreated string       `json:"metadata_created"`
}

// DecodeMetadataInfo derives cryptographic details from vault metadata.
func DecodeMetadataInfo(metadata *domain.VaultMetadata) (*MetadataInfo, []byte, error) {
	if metadata == nil {
		return nil, nil, fmt.Errorf("vault metadata missing")
	}

	params, saltBytes, err := decodeKDFParams(metadata.KDFParams)
	if err != nil {
		return nil, nil, err
	}

	info := &MetadataInfo{
		Cipher:          "AES-256-GCM",
		KDF:             params,
		SaltLength:      len(saltBytes),
		MetadataCreated: metadata.CreatedAt.Format(time.RFC3339),
	}

	return info, saltBytes, nil
}

func decodeKDFParams(raw map[string]interface{}) (Argon2Params, []byte, error) {
	if raw == nil {
		return Argon2Params{}, nil, fmt.Errorf("missing KDF parameters")
	}

	// Attempt flexible decoding via JSON to handle f64 map values.
	serialized, err := json.Marshal(raw)
	if err != nil {
		return Argon2Params{}, nil, fmt.Errorf("failed to marshal KDF params: %w", err)
	}

	var decoded struct {
		Memory      uint32 `json:"memory"`
		Iterations  uint32 `json:"iterations"`
		Parallelism uint8  `json:"parallelism"`
		Salt        string `json:"salt"`
	}

	if err := json.Unmarshal(serialized, &decoded); err != nil {
		return Argon2Params{}, nil, fmt.Errorf("failed to decode KDF params: %w", err)
	}

	saltBytes, err := base64.StdEncoding.DecodeString(decoded.Salt)
	if err != nil {
		return Argon2Params{}, nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	params := Argon2Params{
		Memory:      decoded.Memory,
		Iterations:  decoded.Iterations,
		Parallelism: decoded.Parallelism,
	}

	return params, saltBytes, nil
}
