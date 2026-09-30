package model

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// hashPassword 生成加盐 SHA-256 口令摘要，格式：salt$hash
func hashPassword(password string) (string, error) {
	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", err
	}
	salt := hex.EncodeToString(saltBytes)
	return salt + "$" + hashWithSalt(password, salt), nil
}

func hashWithSalt(password, salt string) string {
	sum := sha256.Sum256([]byte(salt + ":" + password))
	return hex.EncodeToString(sum[:])
}

// compareHash 校验明文口令与摘要是否匹配
func compareHash(stored, password string) error {
	parts := strings.SplitN(stored, "$", 2)
	if len(parts) != 2 {
		return errors.New("invalid password hash format")
	}
	want := []byte(hashWithSalt(password, parts[0]))
	got := []byte(parts[1])
	if subtle.ConstantTimeCompare(want, got) != 1 {
		return fmt.Errorf("incorrect password")
	}
	return nil
}
