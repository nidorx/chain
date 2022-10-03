package chain

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
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
