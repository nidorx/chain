package chain

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"hash/crc32"

	"github.com/cespare/xxhash/v2"
	"github.com/segmentio/ksuid"
)

type Serializer interface {
	Encode(v any) ([]byte, error)
	Decode(data []byte, v any) (any, error)
}

type JsonSerializer struct {
}

func (s *JsonSerializer) Encode(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (s *JsonSerializer) Decode(data []byte, v any) (any, error) {
	if err := json.Unmarshal(data, v); err != nil {
		return nil, err
	}
	return v, nil
}

// HashMD5 computing the MD5 checksum of strings
func HashMD5(text string) string {
	h := md5.New()
	h.Write([]byte(text))
	return hex.EncodeToString(h.Sum(nil))
}

var crc32iSCSI = crc32.MakeTable(crc32.Castagnoli)

func HashCrc32(content []byte) string {
	h := crc32.New(crc32iSCSI)
	h.Write(content)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// Xxh64 return a base64-encoded checksum of a resource using Xxh64 algorithm
//
// Encoded using Base64 URLSafe
func HashXxh64(content []byte) string {
	h := xxhash.New()
	h.Write(content)
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func NewUID() (uid string) {
	return ksuid.New().String()
}
