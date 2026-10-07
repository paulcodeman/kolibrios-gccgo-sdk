package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"kos"
	"os"
)

func main() {
	if n, err := rand.Read(nil); n != 0 || err != nil {
		panic("empty random request")
	}
	first := bytes.Repeat([]byte{0xaa}, 17)
	n, err := rand.Read(first)
	if os.Getenv("KOLIBRI_TEST_EXPECT_NO_ENTROPY") == "1" {
		if n != 0 || err == nil || !bytes.Equal(first, bytes.Repeat([]byte{0xaa}, 17)) {
			panic("missing entropy must return an error without clock fallback")
		}
		kos.DebugString("OPENCODE_SECURE_RANDOM_UNAVAILABLE_PASS\n")
		return
	}
	if n != len(first) || err != nil {
		panic("CPU random read")
	}
	second := make([]byte, 17)
	if n, err := rand.Read(second); n != len(second) || err != nil {
		panic("CPU random tail read")
	}
	if bytes.Equal(first, second) {
		panic("repeated hardware random stream")
	}
	// Key generation/signature verification exercises an actual cryptographic
	// caller. The equality check is not a statistical proof of entropy quality.
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic("cryptographic key generation")
	}
	message := []byte("KolibriOS upstream crypto with CPU entropy")
	signature := ed25519.Sign(private, message)
	if !ed25519.Verify(public, message, signature) {
		panic("signature verification")
	}
	kos.DebugString("OPENCODE_SECURE_RANDOM_PASS\n")
}
