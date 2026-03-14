package util

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"io"
	"slices"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

const HashSha256Prefix = "SHA256:"

func RecursiveEncode(input any, buf *bytes.Buffer) {
	switch v := input.(type) {
	case proto.Message:
		b, err := proto.MarshalOptions{
			Deterministic: true,
		}.Marshal(v)
		if err == nil {
			buf.Write(b)
		} else {
			buf.WriteString(prototext.Format(v))
		}
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}

		slices.Sort(keys)

		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 && i < len(v)-1 {
				buf.WriteByte(',')
			}
			buf.WriteString(`"` + k + `":`)
			RecursiveEncode(v[k], buf)
		}
		buf.WriteByte('}')
	case []any:
		buf.WriteByte('[')
		for i, elem := range v {
			if i > 0 && i < len(v)-1 {
				buf.WriteByte(',')
			}
			RecursiveEncode(elem, buf)
		}
		buf.WriteByte(']')
	case string:
		buf.WriteString(`"` + v + `"`)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		fmt.Fprintf(buf, "%d", v)
	case float32, float64:
		fmt.Fprintf(buf, "%g", v)
	case bool:
		if v {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case nil:
		buf.WriteString("null")
	default:
		fmt.Fprintf(buf, "%v", v)
	}
}

func AnyHash(data any) []byte {
	var buf bytes.Buffer
	RecursiveEncode(data, &buf)
	hash := sha256.Sum256(buf.Bytes())
	return hash[:]
}

// Hash returns a hex-encoded FNV-1a 64-bit hash of the input string.
// This is a fast, non-cryptographic hash suitable for identifiers, cache keys,
// and other use cases where cryptographic security is not required.
// For cryptographic use cases, use HashSHA256 instead.
func Hash(s string) string {
	h := fnv.New64a()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// HashShort returns the first 12 hex characters of the FNV-1a hash.
// Useful for shorter identifiers while maintaining good collision resistance.
func HashShort(s string) string {
	h := fnv.New64a()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))[:12]
}

// HashShort returns the first N hex characters of the FNV-1a hash.
// Useful for shorter identifiers while maintaining good collision resistance.
func HashShortN(s string, n int) string {
	if n <= 0 {
		n = 1
	}

	h := fnv.New64a()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))[:n]
}

// HashSHA256 returns a hex-encoded SHA256 hash of the input string.
// Use this for cryptographic purposes like password hashing, signatures, etc.
func HashSHA256(s string) string {
	hash := sha256.Sum256([]byte(s))
	return HashSha256Prefix + hex.EncodeToString(hash[:])
}

// HashSHA256Short returns the first 12 hex characters of the SHA256 hash.
func HashSHA256Short(s string) string {
	hash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hash[:6]) // 6 bytes = 12 hex chars
}

// HashSHA256Reader computes the SHA256 hash of data read from the provided io.Reader.
func HashSHA256Reader(reader io.Reader) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}
	return HashSha256Prefix + hex.EncodeToString(hash.Sum(nil)), nil
}
