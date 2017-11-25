package sock5

import (
	"crypto/md5"
	"math/rand"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func GenerateAddrHash(addr string) string {
	h := md5.New()

	rawByte := []byte(addr)
	return string(h.Sum(rawByte))
}
