package identity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

// CanonicalizeJSON converts a value into deterministic JSON by sorting object keys.
func CanonicalizeJSON(v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal canonical input: %w", err)
	}

	var normalized interface{}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&normalized); err != nil {
		return nil, fmt.Errorf("decode canonical input: %w", err)
	}

	var buf bytes.Buffer
	if err := writeCanonicalJSON(&buf, normalized); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func writeCanonicalJSON(buf *bytes.Buffer, v interface{}) error {
	switch value := v.(type) {
	case nil:
		buf.WriteString("null")
	case bool:
		if value {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case string:
		encoded, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("marshal string: %w", err)
		}
		buf.Write(encoded)
	case json.Number:
		buf.WriteString(normalizeJSONNumber(value.String()))
	case float64:
		buf.WriteString(normalizeJSONNumber(strconv.FormatFloat(value, 'f', -1, 64)))
	case []interface{}:
		buf.WriteByte('[')
		for i, item := range value {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeCanonicalJSON(buf, item); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	case map[string]interface{}:
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		buf.WriteByte('{')
		for i, key := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			encodedKey, err := json.Marshal(key)
			if err != nil {
				return fmt.Errorf("marshal object key: %w", err)
			}
			buf.Write(encodedKey)
			buf.WriteByte(':')
			if err := writeCanonicalJSON(buf, value[key]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	default:
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("marshal fallback type %T: %w", value, err)
		}
		var fallback interface{}
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		if err := decoder.Decode(&fallback); err != nil {
			return fmt.Errorf("decode fallback type %T: %w", value, err)
		}
		return writeCanonicalJSON(buf, fallback)
	}

	return nil
}

func normalizeJSONNumber(input string) string {
	if input == "" {
		return "0"
	}
	return input
}
