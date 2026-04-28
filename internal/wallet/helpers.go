package wallet

import (
	"fmt"
	"time"
)

// generateWalletID generates a unique wallet ID
func generateWalletID(address string) string {
	return "wallet-" + address
}

// getCurrentTime gets current time
func getCurrentTime() time.Time {
	return time.Now()
}

// ValidatePassword validates password strength
func ValidatePassword(password string) error {
	if password == "" {
		return fmt.Errorf("Password cannot be empty")
	}

	if len(password) < 8 {
		return fmt.Errorf("Password must be at least 8 characters")
	}

	hasLetter := false
	hasDigit := false
	for _, char := range password {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') {
			hasLetter = true
		}
		if char >= '0' && char <= '9' {
			hasDigit = true
		}
	}

	if !hasLetter {
		return fmt.Errorf("Password must contain at least one letter")
	}

	if !hasDigit {
		return fmt.Errorf("Password must contain at least one digit")
	}

	return nil
}
