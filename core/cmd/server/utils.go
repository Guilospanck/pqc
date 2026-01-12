package main

import (
	"math/rand"
)

func GetRandomName() []byte {
	idx := rand.Intn(len(RANDOM_NAMES))
	return []byte(RANDOM_NAMES[idx])
}
