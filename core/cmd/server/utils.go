package main

import (
	"math/rand"
)

func GetRandomName() []byte {
	idx := rand.Intn(len(RANDOM_NAMES))
	return []byte(RANDOM_NAMES[idx])
}

func GetRandomColor() []byte {
	idx := rand.Intn(len(RANDOM_COLORS))
	return []byte(RANDOM_COLORS[idx])
}
