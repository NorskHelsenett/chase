package utils

import (
	"crypto/rand"
	"log"
	"math/big"
	"os"
)

func GetEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = fallback
	}
	return value
}

func GenerateAPIToken() string {
	// Draw uniformly from an alphanumeric alphabet so the result is always
	// exactly 32 characters ("sk-" + 32 = 35, which auth relies on). The
	// previous base64-and-strip approach could yield fewer than 32 chars and
	// panic on the slice.
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	out := make([]byte, 32)
	for i := range out {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			log.Fatalf("Error generating random bytes: %v", err)
		}
		out[i] = alphabet[n.Int64()]
	}
	return "sk-" + string(out)
}
