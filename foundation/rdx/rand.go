package rdx

import "crypto/rand"

// Indirection so tests can stub the random source if ever needed.
var cryptoReader = rand.Reader
