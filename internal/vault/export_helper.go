package vault

import (
	"encoding/json"
)

// IsEncryptedExport checks if the export data is encrypted
func IsEncryptedExport(data []byte) bool {
	var format ExportFormat
	if err := json.Unmarshal(data, &format); err != nil {
		return false
	}
	return format.Encrypted
}
