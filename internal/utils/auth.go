package utils

import (
	"crypto/sha1"
	"fmt"
	"strings"
)

const (
	salt = ";knmmm3rjoq; 2vr541jdhaDCGV1UE9PED"

	en_lower = "abcdefghijklmnopqrstuvwxyz"
	en_upper = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits   = "0123456789"
	symbols  = "@_-"
)

func ValidateLogin(login string) error {
	en_letter_contain := false
	for _, elem := range login {
		if strings.Contains(en_lower+en_upper, string(elem)) {
			en_letter_contain = true
			continue
		} else if strings.Contains(digits, string(elem)) {
			continue
		}
		return fmt.Errorf("Invalid login format: %s", login)
	}
	if len(login) < 8 {
		return fmt.Errorf("Login must at least 8 characters")
	}
	if !(en_letter_contain) {
		return fmt.Errorf("Login must contain at least 1 letter from English")
	}
	return nil
}

func ValidatePassword(password string) error {
	for _, elem := range password {
		if !strings.Contains(en_lower+en_upper+digits+symbols, string(elem)) {
			return fmt.Errorf("Invalid password format: %s", password)
		}
	}
	if len(password) < 8 {
		return fmt.Errorf("Password must at least 8 characters")
	}
	return nil
}
func GeneratePasswordHash(password string) string {
	hash := sha1.New()

	hash.Write([]byte(password))
	return fmt.Sprintf("%x", hash.Sum([]byte(salt)))
}
