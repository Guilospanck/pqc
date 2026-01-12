package main

import (
	"math/rand"
)

func GetRandomColor() []byte {
	idx := rand.Intn(len(RANDOM_COLORS))
	return []byte(RANDOM_COLORS[idx])
}
