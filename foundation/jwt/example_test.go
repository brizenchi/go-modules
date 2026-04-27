package jwt_test

import (
	"errors"
	"fmt"
	"time"

	"github.com/brizenchi/go-modules/foundation/jwt"
)

// Example_hS256 shows the round-trip: build a signer, sign claims,
// parse them back. HS256 is the right choice when the same service
// signs and verifies (or all parties share a secret).
func Example_hS256() {
	signer, err := jwt.NewHS256("super-secret-32-bytes-of-entropy!", jwt.Options{
		Issuer:   "auth-svc",
		Audience: []string{"api"},
		Leeway:   30 * time.Second,
	})
	if err != nil {
		panic(err)
	}

	token, err := signer.Sign(jwt.Claims{
		Subject: "user-42",
		TTL:     time.Hour,
		Extra:   map[string]any{"role": "admin"},
	})
	if err != nil {
		panic(err)
	}

	parsed, err := signer.Parse(token)
	if err != nil {
		panic(err)
	}
	role, _ := parsed.Extra["role"].(string)
	fmt.Println(parsed.Subject, role)
	// Output: user-42 admin
}

// Example_expired shows how to discriminate the expired case from
// other parse errors. Use errors.Is — never string matching.
func Example_expired() {
	signer, err := jwt.NewHS256("super-secret-32-bytes-of-entropy!", jwt.Options{})
	if err != nil {
		panic(err)
	}
	token, _ := signer.Sign(jwt.Claims{Subject: "u", TTL: 1 * time.Nanosecond})

	time.Sleep(10 * time.Millisecond)
	_, err = signer.Parse(token)
	if errors.Is(err, jwt.ErrExpired) {
		fmt.Println("expired")
	}
	// Output: expired
}
