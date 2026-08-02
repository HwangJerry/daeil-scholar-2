package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

var ErrCredentialVaultNotConfigured = errors.New("social credential vault is not configured")

type SocialCredentialVault struct {
	aead cipher.AEAD
}

func NewSocialCredentialVault(encodedKey string) (*SocialCredentialVault, error) {
	if encodedKey == "" {
		return &SocialCredentialVault{}, nil
	}
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		return nil, errors.New("SOCIAL_CREDENTIAL_ENCRYPTION_KEY must be base64")
	}
	if len(key) != 32 {
		return nil, errors.New("SOCIAL_CREDENTIAL_ENCRYPTION_KEY must decode to 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &SocialCredentialVault{aead: aead}, nil
}

func (v *SocialCredentialVault) Ready() bool {
	return v != nil && v.aead != nil
}

func (v *SocialCredentialVault) Encrypt(value string) (string, error) {
	if v == nil || v.aead == nil {
		return "", ErrCredentialVaultNotConfigured
	}
	nonce := make([]byte, v.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := v.aead.Seal(nonce, nonce, []byte(value), nil)
	return base64.RawStdEncoding.EncodeToString(sealed), nil
}

func (v *SocialCredentialVault) Decrypt(value string) (string, error) {
	if v == nil || v.aead == nil {
		return "", ErrCredentialVaultNotConfigured
	}
	encoded, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	nonceSize := v.aead.NonceSize()
	if len(encoded) < nonceSize {
		return "", errors.New("encrypted credential is malformed")
	}
	plain, err := v.aead.Open(nil, encoded[:nonceSize], encoded[nonceSize:], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
