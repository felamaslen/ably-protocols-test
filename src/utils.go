package main

import (
	"math"
	"math/rand"
)

func generateInitialNumber() uint32 {
	return uint32(math.Max(1, rand.Float64()*0xff))
}
