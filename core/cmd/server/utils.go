package main

import (
	"math/rand"
)

func GetRandomName() string {
	idx := rand.Intn(len(RANDOM_NAMES))
	return RANDOM_NAMES[idx]
}

func GetRandomColor() string {
	idx := rand.Intn(len(RANDOM_COLORS))
	return RANDOM_COLORS[idx]
}
