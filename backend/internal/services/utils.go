package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"
)

func RandomCode(prefix string, bytes int) string {
	buf := make([]byte, bytes)
	_, _ = rand.Read(buf)
	return prefix + hex.EncodeToString(buf)
}

func OrderNo() string {
	return fmt.Sprintf("SL%s%s", time.Now().Format("20060102150405"), RandomDigits(4))
}

func PickupCode() string {
	return RandomDigits(4)
}

func RandomDigits(length int) string {
	out := make([]byte, length)
	for i := range out {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			out[i] = byte('0' + i%10)
			continue
		}
		out[i] = byte('0' + n.Int64())
	}
	return string(out)
}
