package internal

import (
	"math/rand"
)

func shuffle(arr []string) {
	for i := range arr {
		j := rand.Intn(i + 1)
		arr[i], arr[j] = arr[j], arr[i]
	}
}
