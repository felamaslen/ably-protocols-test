package main

import (
	"math/rand"
)

func generateInitialNumber() uint32 {
	return uint32(1 + rand.Uint32()%(0xff-1))
}
