// account_erasure_context_cipher.go — Authenticated, request-bound temporary deletion context.
package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/dflh-saf/backend/internal/model"
)

type ErasureContextCipher struct{ aead cipher.AEAD }

func NewErasureContextCipher(keyHex string) (*ErasureContextCipher, error) {
	key, err := hex.DecodeString(keyHex)
	if err != nil || len(key) != 32 {
		return nil, &model.ErasureBlocked{Code: "ERASURE_CONTEXT_KEY_REQUIRED"}
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &ErasureContextCipher{aead: aead}, nil
}
func contextAAD(id int64) []byte { return []byte(fmt.Sprintf("DFLH_ERASURE_CONTEXT_V1:%d", id)) }
func (c *ErasureContextCipher) Seal(s model.ErasureExternalSubject) ([]byte, error) {
	if c == nil {
		return nil, &model.ErasureBlocked{Code: "ERASURE_CONTEXT_KEY_REQUIRED"}
	}
	plain, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, err
	}
	return c.aead.Seal(nonce, nonce, plain, contextAAD(s.RequestID)), nil
}
func (c *ErasureContextCipher) Open(w model.ErasureWork, data []byte) (model.ErasureExternalSubject, error) {
	var s model.ErasureExternalSubject
	blocked := &model.ErasureBlocked{Code: "ERASURE_CONTEXT_UNREADABLE"}
	if c == nil {
		return s, &model.ErasureBlocked{Code: "ERASURE_CONTEXT_KEY_REQUIRED"}
	}
	n := c.aead.NonceSize()
	if len(data) < n+c.aead.Overhead() {
		return s, blocked
	}
	plain, err := c.aead.Open(nil, data[:n], data[n:], contextAAD(w.RequestID))
	if err != nil {
		return s, blocked
	}
	if json.Unmarshal(plain, &s) != nil || s.RequestID != w.RequestID || s.UserSeq != w.UserSeq {
		return model.ErasureExternalSubject{}, blocked
	}
	return s, nil
}
