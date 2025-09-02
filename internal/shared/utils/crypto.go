package utils

import (
	"crypto/rand"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
)

// GenerateSalt génère un salt aléatoire de 32 bytes (64 caractères hex)
func GenerateSalt() (string, error) {
	saltBytes := make([]byte, 32)
	_, err := rand.Read(saltBytes)
	if err != nil {
		return "", fmt.Errorf("impossible de générer le salt: %w", err)
	}

	return hex.EncodeToString(saltBytes), nil
}

// HashPasswordSHA512 hash un mot de passe avec SHA512 et un salt
func HashPasswordSHA512(password, salt string) string {
	// Concaténer le mot de passe et le salt
	combined := password + salt

	// Calculer le hash SHA512
	hasher := sha512.New()
	hasher.Write([]byte(combined))
	hashedBytes := hasher.Sum(nil)

	// Retourner le hash en hexadécimal
	return hex.EncodeToString(hashedBytes)
}

// VerifyPasswordSHA512 vérifie un mot de passe contre un hash SHA512
func VerifyPasswordSHA512(password, salt, hashedPassword string) bool {
	// Recalculer le hash avec le mot de passe fourni et le salt
	calculatedHash := HashPasswordSHA512(password, salt)

	// Comparer les hashs
	return calculatedHash == hashedPassword
}

// ValidateSalt vérifie que le salt a le bon format (64 caractères hex)
func ValidateSalt(salt string) error {
	if len(salt) != 64 {
		return fmt.Errorf("salt invalide: doit faire 64 caractères (32 bytes hex)")
	}

	// Vérifier que c'est bien de l'hex
	_, err := hex.DecodeString(salt)
	if err != nil {
		return fmt.Errorf("salt invalide: doit être au format hexadécimal")
	}

	return nil
}

// ValidateHashedPassword vérifie que le hash a le bon format (128 caractères hex pour SHA512)
func ValidateHashedPassword(hashedPassword string) error {
	if len(hashedPassword) != 128 {
		return fmt.Errorf("hash mot de passe invalide: doit faire 128 caractères (64 bytes SHA512 hex)")
	}

	// Vérifier que c'est bien de l'hex
	_, err := hex.DecodeString(hashedPassword)
	if err != nil {
		return fmt.Errorf("hash mot de passe invalide: doit être au format hexadécimal")
	}

	return nil
}
