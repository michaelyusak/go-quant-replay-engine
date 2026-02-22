package common

import "math/rand/v2"

func CreateRandomString(targetLen int) string {
	pool := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	lenPool := len(pool)
	b := make([]byte, targetLen)

	for i := range b {
		b[i] = pool[rand.IntN(lenPool)]
	}

	return string(b)
}
