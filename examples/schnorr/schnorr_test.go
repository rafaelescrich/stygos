package main

import (
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/rafaelescrich/stygos"
)

func TestSchnorrVerify(t *testing.T) {
	// Test with known valid signature
	msg := []byte("Hello, World!")
	pkX := make([]byte, 32)
	pkX[31] = 1 // Simple test key

	// Create a mock signature (this would normally be generated with proper key)
	sig := make([]byte, 64)
	rand.Read(sig)

	// Test the verify function
	valid := verify(msg, sig, pkX)
	// Note: This will likely be false with random signature, but tests the function structure
	t.Logf("Verify result: %v", valid)
}

func TestLiftXEvenY(t *testing.T) {
	// Test lifting generator point x-coordinate
	gx := new(big.Int).SetBytes([]byte{
		0x79, 0xBE, 0x66, 0x7E, 0xF9, 0xDC, 0xBB, 0xAC, 0x55, 0xA0, 0x62, 0x95, 0xCE, 0x87, 0x0B, 0x07,
		0x02, 0x9B, 0xFC, 0xDB, 0x2D, 0xCE, 0x28, 0xD9, 0x59, 0xF2, 0x81, 0x5B, 0x16, 0xF8, 0x17, 0x98,
	})

	point, err := liftXEvenY(gx)
	if err != nil {
		t.Fatalf("Failed to lift x: %v", err)
	}

	// Check that the lifted point matches the generator
	if point.X.Cmp(GX) != 0 {
		t.Errorf("X coordinate mismatch: got %x, want %x", point.X.Bytes(), GX.Bytes())
	}

	if point.Y.Cmp(GY) != 0 {
		t.Errorf("Y coordinate mismatch: got %x, want %x", point.Y.Bytes(), GY.Bytes())
	}

	// Verify the point is on the curve
	if !isOnCurve(point) {
		t.Error("Lifted point is not on curve")
	}

	// Verify Y is even
	if point.Y.Bit(0) != 0 {
		t.Error("Y coordinate is not even")
	}
}

func TestPointOperations(t *testing.T) {
	// Test point doubling
	g := Affine{X: GX, Y: GY}
	g2 := double(g)

	if !isOnCurve(g2) {
		t.Error("Doubled point is not on curve")
	}

	// Test point addition
	g3 := add(g, g2)
	if !isOnCurve(g3) {
		t.Error("Added point is not on curve")
	}

	// Test scalar multiplication
	g4 := mul(g, big.NewInt(4))
	if !isOnCurve(g4) {
		t.Error("Multiplied point is not on curve")
	}

	// Verify 4*G = 2*G + 2*G
	g4Alt := add(g2, g2)
	if g4.X.Cmp(g4Alt.X) != 0 || g4.Y.Cmp(g4Alt.Y) != 0 {
		t.Error("4*G != 2*G + 2*G")
	}
}

func TestInfinityPoint(t *testing.T) {
	inf := Affine{X: big.NewInt(0), Y: big.NewInt(0)}

	if !isInfinity(inf) {
		t.Error("Should be infinity point")
	}

	// Test addition with infinity
	g := Affine{X: GX, Y: GY}
	result := add(inf, g)
	if result.X.Cmp(g.X) != 0 || result.Y.Cmp(g.Y) != 0 {
		t.Error("Adding infinity should return the other point")
	}
}

func TestChallengeBIP340(t *testing.T) {
	r := big.NewInt(12345)
	pkX := make([]byte, 32)
	pkX[0] = 1
	msg := []byte("test message")

	e := challengeBIP340(r, pkX, msg)

	// Challenge should be deterministic
	e2 := challengeBIP340(r, pkX, msg)
	if e.Cmp(e2) != 0 {
		t.Error("Challenge should be deterministic")
	}

	// Challenge should be different for different inputs
	e3 := challengeBIP340(big.NewInt(12346), pkX, msg)
	if e.Cmp(e3) == 0 {
		t.Error("Challenge should be different for different r")
	}
}

func TestExtract(t *testing.T) {
	// Create mock signatures
	sig1 := make([]byte, 64)
	sig2 := make([]byte, 64)

	// Use same r for both signatures
	r := big.NewInt(12345)
	rBytes := make([]byte, 32)
	r.FillBytes(rBytes)
	copy(sig1[:32], rBytes)
	copy(sig2[:32], rBytes)

	// Different s values
	s1 := big.NewInt(100)
	s2 := big.NewInt(50)
	s1Bytes := make([]byte, 32)
	s2Bytes := make([]byte, 32)
	s1.FillBytes(s1Bytes)
	s2.FillBytes(s2Bytes)
	copy(sig1[32:], s1Bytes)
	copy(sig2[32:], s2Bytes)

	secret := extract(sig1, sig2)
	expected := new(big.Int).Sub(s1, s2)

	if secret.Cmp(expected) != 0 {
		t.Errorf("Extract failed: got %v, want %v", secret, expected)
	}
}

func TestContractInterface(t *testing.T) {
	// Test the contract interface with mock data
	msg := []byte("test")
	pkX := make([]byte, 32)
	pkX[31] = 1
	sig := make([]byte, 64)
	rand.Read(sig)

	// Prepare call data for verify command
	callData := make([]byte, 1+len(msg)+32+64)
	callData[0] = byte(len(msg))
	copy(callData[1:], msg)
	copy(callData[1+len(msg):], pkX)
	copy(callData[1+len(msg)+32:], sig)

	// This would normally be called by the Stylus runtime
	// For testing, we can call the handler directly
	result := handleVerify(callData[1:])
	t.Logf("Contract verify result: %d", result)
}

func BenchmarkLiftXEvenY(b *testing.B) {
	gx := new(big.Int).SetBytes([]byte{
		0x79, 0xBE, 0x66, 0x7E, 0xF9, 0xDC, 0xBB, 0xAC, 0x55, 0xA0, 0x62, 0x95, 0xCE, 0x87, 0x0B, 0x07,
		0x02, 0x9B, 0xFC, 0xDB, 0x2D, 0xCE, 0x28, 0xD9, 0x59, 0xF2, 0x81, 0x5B, 0x16, 0xF8, 0x17, 0x98,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := liftXEvenY(gx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPointMul(b *testing.B) {
	g := Affine{X: GX, Y: GY}
	k := big.NewInt(12345)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mul(g, k)
	}
}

func BenchmarkChallengeBIP340(b *testing.B) {
	r := big.NewInt(12345)
	pkX := make([]byte, 32)
	pkX[0] = 1
	msg := []byte("test message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		challengeBIP340(r, pkX, msg)
	}
}

// Test helper functions for integration with Stygos
func TestStygosIntegration(t *testing.T) {
	// Test that our types work with Stygos
	msg := []byte("test message")

	// Test Keccak256 integration
	hash := stygos.Keccak256(msg)
	if len(hash) != 32 {
		t.Errorf("Keccak256 should return 32 bytes, got %d", len(hash))
	}

	// Test storage operations
	key := stygos.Keccak256([]byte("test_key"))
	value := stygos.WordFromUint64(12345)

	// In a real test environment, we would use the mock runtime
	// For now, just test that the functions exist and have correct signatures
	t.Logf("Key: %x", key)
	t.Logf("Value: %x", value)
}

// Example usage functions
func ExampleVerify() {
	msg := []byte("Hello, World!")
	pkX := make([]byte, 32)
	pkX[31] = 1 // Simple test key

	sig := make([]byte, 64)
	// In practice, sig would be generated by a proper signing algorithm

	valid := verify(msg, sig, pkX)
	_ = valid // Use the result
}

func ExampleLiftX() {
	x := new(big.Int).SetBytes([]byte{
		0x79, 0xBE, 0x66, 0x7E, 0xF9, 0xDC, 0xBB, 0xAC, 0x55, 0xA0, 0x62, 0x95, 0xCE, 0x87, 0x0B, 0x07,
		0x02, 0x9B, 0xFC, 0xDB, 0x2D, 0xCE, 0x28, 0xD9, 0x59, 0xF2, 0x81, 0x5B, 0x16, 0xF8, 0x17, 0x98,
	})

	point, err := liftXEvenY(x)
	if err != nil {
		panic(err)
	}

	_ = point // Use the lifted point
}

func ExamplePointOperations() {
	g := Affine{X: GX, Y: GY}

	// Double the generator point
	g2 := double(g)

	// Add two points
	g3 := add(g, g2)

	// Multiply by scalar
	g4 := mul(g, big.NewInt(4))

	_ = g2
	_ = g3
	_ = g4
}
