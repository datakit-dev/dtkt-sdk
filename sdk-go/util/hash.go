package util

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"hash/fnv"
	"io"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

const HashSha256Prefix = "SHA256:"

func HashAny(value any) (string, error) {
	b, err := HashAnyBytes(value)
	if err != nil {
		return "", err
	}
	return HashSha256Prefix + hex.EncodeToString(b), nil
}

func HashAnyBytes(value any) (_ []byte, err error) {
	var buf bytes.Buffer
	_, err = deterministicBytes(&buf, value)
	if err != nil {
		return
	}

	h := sha256.New()
	_, err = io.Copy(h, &buf)
	if err != nil {
		return
	}

	return h.Sum(nil), nil
}

func HashAnyFunc[V ~string | ~[]byte](value any, hashFunc func(V) string) (string, error) {
	var buf bytes.Buffer
	_, err := deterministicBytes(&buf, value)
	if err != nil {
		return "", err
	}

	return hashFunc(V(buf.Bytes())), nil
}

// Hash returns a hex-encoded FNV-1a 64-bit hash of the input string.
// This is a fast, non-cryptographic hash suitable for identifiers, cache keys,
// and other use cases where cryptographic security is not required.
// For cryptographic use cases, use HashSHA256 instead.
func Hash[V ~string | ~[]byte](s V) string {
	h := fnv.New64a()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// HashShort returns the first 12 hex characters of the FNV-1a hash.
// Useful for shorter identifiers while maintaining good collision resistance.
func HashShort[V ~string | ~[]byte](s V) string {
	h := fnv.New64a()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))[:12]
}

// HashShort returns the first N hex characters of the FNV-1a hash.
// Useful for shorter identifiers while maintaining good collision resistance.
func HashShortN[V ~string | ~[]byte](s V, n int) string {
	if n <= 0 {
		n = 1
	}
	h := fnv.New64a()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))[:n]
}

// HashSHA256 returns a hex-encoded SHA256 hash of the input string.
// Use this for cryptographic purposes like password hashing, signatures, etc.
func HashSHA256[V ~string | ~[]byte](s V) string {
	h := sha256.Sum256([]byte(s))
	return HashSha256Prefix + hex.EncodeToString(h[:])
}

// HashSHA256Short returns the first 12 hex characters of the SHA256 hash.
func HashSHA256Short[V ~string | ~[]byte](s V) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:6]) // 6 bytes = 12 hex chars
}

// HashSHA256Reader computes the SHA256 hash of data read from the provided io.Reader.
func HashSHA256Reader(reader io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, reader); err != nil {
		return "", err
	}
	return HashSha256Prefix + hex.EncodeToString(h.Sum(nil)), nil
}

func deterministicBytes(buf *bytes.Buffer, value any) (int, error) {
	switch value := value.(type) {
	case io.Reader:
		s, err := io.Copy(buf, value)
		if err != nil {
			return 0, err
		}
		return int(s), nil
	case []byte:
		return buf.Write(value)
	case proto.Message:
		b, err := proto.MarshalOptions{
			AllowPartial:  true,
			Deterministic: true,
		}.Marshal(value)
		if err != nil {
			return 0, err
		}

		return buf.Write(b)
	}

	b, err := json.Marshal(value)
	if err != nil {
		return 0, err
	}

	proto := new(structpb.Value)
	err = protojson.Unmarshal(b, proto)
	if err != nil {
		return 0, err
	}

	return deterministicBytes(buf, proto)
}
