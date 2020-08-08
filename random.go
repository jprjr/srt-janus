package main

// copying some internals from pion

import (
    "github.com/pion/randutil"
)

const (
    runesAlpha = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

var globalMathRandomGenerator = randutil.NewMathRandomGenerator()

func MathRandAlpha(n int) string {
        return globalMathRandomGenerator.GenerateString(n, runesAlpha)
}

func RandUint32() uint32 {
        return globalMathRandomGenerator.Uint32()
}
