package id

import (
	"crypto/rand"
	"math/big"
	"regexp"
)

const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
const length = 8

var validRe = regexp.MustCompile(`^[A-Za-z0-9]{8}$`)

func Generate() (string, error) {
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		b[i] = alphabet[n.Int64()]
	}
	return string(b), nil
}

func Valid(s string) bool {
	return validRe.MatchString(s)
}
