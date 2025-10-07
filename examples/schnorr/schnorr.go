package main

import (
	"crypto/sha256"
	"errors"
	"math/big"

	"github.com/rafaelescrich/stygos"
)

// secp256k1 constants
var (
	// Field modulus p
	P = new(big.Int).SetBytes([]byte{
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFE, 0xFF, 0xFF, 0xFC, 0x2F,
	})

	// Curve order n
	N = new(big.Int).SetBytes([]byte{
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFE,
		0xBA, 0xAE, 0xDC, 0xE6, 0xAF, 0x48, 0xA0, 0x3B, 0xBF, 0xD2, 0x5E, 0x8C, 0xD0, 0x36, 0x41, 0x41,
	})

	// Curve parameter b
	B = big.NewInt(7)

	// Generator point G
	GX = new(big.Int).SetBytes([]byte{
		0x79, 0xBE, 0x66, 0x7E, 0xF9, 0xDC, 0xBB, 0xAC, 0x55, 0xA0, 0x62, 0x95, 0xCE, 0x87, 0x0B, 0x07,
		0x02, 0x9B, 0xFC, 0xDB, 0x2D, 0xCE, 0x28, 0xD9, 0x59, 0xF2, 0x81, 0x5B, 0x16, 0xF8, 0x17, 0x98,
	})
	GY = new(big.Int).SetBytes([]byte{
		0x48, 0x3A, 0xDA, 0x77, 0x26, 0xA3, 0xC4, 0x65, 0x5D, 0xA4, 0xFB, 0xFC, 0x0E, 0x11, 0x08, 0xA8,
		0xFD, 0x17, 0xB4, 0x48, 0xA6, 0x85, 0x54, 0x19, 0x9C, 0x47, 0xD0, 0x8F, 0xFB, 0x10, 0xD4, 0xB8,
	})

	// (p+1)/4 for square root in F_p
	SQRT_EXP = func() *big.Int {
		result := new(big.Int).Add(P, big.NewInt(1))
		result.Rsh(result, 2)
		return result
	}()
)

// Error definitions
var (
	ErrInvalidSignatureLength = errors.New("invalid signature length")
	ErrInvalidPubKeyLength    = errors.New("invalid public key length")
	ErrLiftXFailed            = errors.New("lift x failed")
	ErrScalarOutOfRange       = errors.New("scalar out of range")
	ErrInfinityPoint          = errors.New("infinity point")
)

// Affine point representation
type Affine struct {
	X *big.Int
	Y *big.Int
}

// Commands for the contract
const (
	CMD_VERIFY         = 0
	CMD_ADAPTOR_VERIFY = 1
	CMD_EXTRACT        = 2
	CMD_LIFT_X         = 3
	CMD_POINT_ADD      = 4
	CMD_POINT_MUL      = 5
)

//export entrypoint
func entrypoint() int32 {
	callData, err := stygos.GetCallData()
	if err != nil || len(callData) < 1 {
		return 1 // Invalid input
	}

	command := callData[0]
	args := callData[1:]

	switch command {
	case CMD_VERIFY:
		return handleVerify(args)
	case CMD_ADAPTOR_VERIFY:
		return handleAdaptorVerify(args)
	case CMD_EXTRACT:
		return handleExtract(args)
	case CMD_LIFT_X:
		return handleLiftX(args)
	case CMD_POINT_ADD:
		return handlePointAdd(args)
	case CMD_POINT_MUL:
		return handlePointMul(args)
	default:
		return 1 // Unknown command
	}
}

// handleVerify verifies a standard BIP-340 signature
func handleVerify(args []byte) int32 {
	if len(args) < 97 { // 1 + 32 + 64 = 97 bytes minimum
		return 1
	}

	msgLen := int(args[0])
	if len(args) < 1+msgLen+32+64 {
		return 1
	}

	msg := args[1 : 1+msgLen]
	pkX := args[1+msgLen : 1+msgLen+32]
	sig := args[1+msgLen+32 : 1+msgLen+32+64]

	valid := verify(msg, sig, pkX)
	if valid {
		return 0
	}
	return 1
}

// handleAdaptorVerify verifies an adaptor signature
func handleAdaptorVerify(args []byte) int32 {
	if len(args) < 129 { // 1 + 32 + 64 + 32 + 32 = 129 bytes minimum
		return 1
	}

	msgLen := int(args[0])
	if len(args) < 1+msgLen+32+64+32+32 {
		return 1
	}

	msg := args[1 : 1+msgLen]
	pkX := args[1+msgLen : 1+msgLen+32]
	sig := args[1+msgLen+32 : 1+msgLen+32+64]
	tx := args[1+msgLen+32+64 : 1+msgLen+32+64+32]
	ty := args[1+msgLen+32+64+32 : 1+msgLen+32+64+32+32]

	T := Affine{
		X: new(big.Int).SetBytes(tx),
		Y: new(big.Int).SetBytes(ty),
	}

	valid := adaptorVerify(msg, sig, pkX, T)
	if valid {
		return 0
	}
	return 1
}

// handleExtract extracts adaptor secret
func handleExtract(args []byte) int32 {
	if len(args) != 128 { // 64 + 64 = 128 bytes
		return 1
	}

	sig := args[:64]
	adaptorSig := args[64:128]

	secret := extract(sig, adaptorSig)

	// Return the secret as 32 bytes
	result := make([]byte, 32)
	secretBytes := secret.Bytes()
	copy(result[32-len(secretBytes):], secretBytes)
	stygos.SetReturnData(result)

	return 0
}

// handleLiftX lifts x-coordinate to even-Y point
func handleLiftX(args []byte) int32 {
	if len(args) != 32 {
		return 1
	}

	x := new(big.Int).SetBytes(args)
	point, err := liftXEvenY(x)
	if err != nil {
		return 1
	}

	// Return both x and y coordinates (64 bytes total)
	result := make([]byte, 64)
	xBytes := point.X.Bytes()
	yBytes := point.Y.Bytes()
	copy(result[32-len(xBytes):32], xBytes)
	copy(result[64-len(yBytes):], yBytes)
	stygos.SetReturnData(result)

	return 0
}

// handlePointAdd adds two points
func handlePointAdd(args []byte) int32 {
	if len(args) != 128 { // 32 + 32 + 32 + 32 = 128 bytes
		return 1
	}

	p1x := new(big.Int).SetBytes(args[:32])
	p1y := new(big.Int).SetBytes(args[32:64])
	p2x := new(big.Int).SetBytes(args[64:96])
	p2y := new(big.Int).SetBytes(args[96:128])

	p1 := Affine{X: p1x, Y: p1y}
	p2 := Affine{X: p2x, Y: p2y}

	result := add(p1, p2)

	// Return result coordinates (64 bytes total)
	resultBytes := make([]byte, 64)
	xBytes := result.X.Bytes()
	yBytes := result.Y.Bytes()
	copy(resultBytes[32-len(xBytes):32], xBytes)
	copy(resultBytes[64-len(yBytes):], yBytes)
	stygos.SetReturnData(resultBytes)

	return 0
}

// handlePointMul multiplies a point by a scalar
func handlePointMul(args []byte) int32 {
	if len(args) != 96 { // 32 + 32 + 32 = 96 bytes
		return 1
	}

	px := new(big.Int).SetBytes(args[:32])
	py := new(big.Int).SetBytes(args[32:64])
	k := new(big.Int).SetBytes(args[64:96])

	point := Affine{X: px, Y: py}
	result := mul(point, k)

	// Return result coordinates (64 bytes total)
	resultBytes := make([]byte, 64)
	xBytes := result.X.Bytes()
	yBytes := result.Y.Bytes()
	copy(resultBytes[32-len(xBytes):32], xBytes)
	copy(resultBytes[64-len(yBytes):], yBytes)
	stygos.SetReturnData(resultBytes)

	return 0
}

// verify verifies a standard BIP-340 signature
func verify(msg, sig, pkX []byte) bool {
	if len(sig) != 64 || len(pkX) != 32 {
		return false
	}

	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:])

	if r.Cmp(P) >= 0 || s.Cmp(N) >= 0 {
		return false
	}

	pk, err := liftXEvenY(new(big.Int).SetBytes(pkX))
	if err != nil {
		return false
	}

	// e = H_tag(bytes32(r) || bytes(P) || m) mod n
	e := challengeBIP340(r, pkX, msg)
	e.Mod(e, N)

	// Compute R = s*G - e*P
	sG := mul(Affine{X: GX, Y: GY}, s)
	eP := mul(pk, e)
	negEP := Affine{X: eP.X, Y: new(big.Int).Sub(P, eP.Y)}
	R := add(sG, negEP)

	if isInfinity(R) {
		return false
	}

	// Require even Y and x(R) == r
	return R.Y.Bit(0) == 0 && R.X.Cmp(r) == 0
}

// adaptorVerify verifies an adaptor signature
func adaptorVerify(msg, sig, pkX []byte, T Affine) bool {
	if len(sig) != 64 || len(pkX) != 32 {
		return false
	}

	r := new(big.Int).SetBytes(sig[:32])
	sPrime := new(big.Int).SetBytes(sig[32:])

	if r.Cmp(P) >= 0 || sPrime.Cmp(N) >= 0 {
		return false
	}

	pk, err := liftXEvenY(new(big.Int).SetBytes(pkX))
	if err != nil {
		return false
	}

	if !isOnCurve(T) {
		return false
	}

	// R' = R + T
	R, err := liftXEvenY(r)
	if err != nil {
		return false
	}
	Rp := add(R, T)

	// Challenge uses bytes(R') || pk || m
	e := challengeBIP340(Rp.X, pkX, msg)
	e.Mod(e, N)

	// Check s'·G == R + e·P
	sG := mul(Affine{X: GX, Y: GY}, sPrime)
	eP := mul(pk, e)
	rhs := add(R, eP)

	if isInfinity(sG) || isInfinity(rhs) {
		return false
	}

	// Compute implied R* = s'G - eP
	negEP := Affine{X: eP.X, Y: new(big.Int).Sub(P, eP.Y)}
	Rstar := add(sG, negEP)
	if isInfinity(Rstar) {
		return false
	}

	return Rstar.Y.Bit(0) == 0 && Rstar.X.Cmp(r) == 0
}

// extract extracts adaptor secret t = (s - s') mod n
func extract(sig, adaptorSig []byte) *big.Int {
	if len(sig) != 64 || len(adaptorSig) != 64 {
		return big.NewInt(0)
	}

	r1 := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:])
	r2 := new(big.Int).SetBytes(adaptorSig[:32])
	sPrime := new(big.Int).SetBytes(adaptorSig[32:])

	if r1.Cmp(r2) != 0 {
		return big.NewInt(0)
	}

	// (s - s') mod n
	if s.Cmp(sPrime) >= 0 {
		return new(big.Int).Sub(s, sPrime)
	}
	return new(big.Int).Sub(N, new(big.Int).Sub(sPrime, s))
}

// challengeBIP340 computes BIP-340 challenge hash
func challengeBIP340(r *big.Int, pkX, msg []byte) *big.Int {
	// Precompute tag hash
	tagHash := sha256.Sum256([]byte("BIP0340/challenge"))

	// Create tagged hash: SHA256(SHA256(tag)||SHA256(tag)||data)
	rBytes := make([]byte, 32)
	r.FillBytes(rBytes)

	data := append(rBytes, pkX...)
	data = append(data, msg...)

	h := sha256.Sum256(append(append(tagHash[:], tagHash[:]...), data...))
	return new(big.Int).SetBytes(h[:])
}

// isOnCurve checks if a point is on the curve
func isOnCurve(p Affine) bool {
	if isInfinity(p) {
		return true
	}

	yy := new(big.Int).Mul(p.Y, p.Y)
	yy.Mod(yy, P)

	xx := new(big.Int).Mul(p.X, p.X)
	xx.Mod(xx, P)

	xxx := new(big.Int).Mul(xx, p.X)
	xxx.Mod(xxx, P)

	rhs := new(big.Int).Add(xxx, B)
	rhs.Mod(rhs, P)

	return yy.Cmp(rhs) == 0
}

// isInfinity checks if a point is at infinity
func isInfinity(p Affine) bool {
	return p.X.Cmp(big.NewInt(0)) == 0 && p.Y.Cmp(big.NewInt(0)) == 0
}

// add adds two points
func add(p1, p2 Affine) Affine {
	if isInfinity(p1) {
		return p2
	}
	if isInfinity(p2) {
		return p1
	}

	if p1.X.Cmp(p2.X) == 0 {
		sum := new(big.Int).Add(p1.Y, p2.Y)
		sum.Mod(sum, P)
		if p1.Y.Cmp(big.NewInt(0)) == 0 || sum.Cmp(big.NewInt(0)) == 0 {
			return Affine{X: big.NewInt(0), Y: big.NewInt(0)}
		}
		return double(p1)
	}

	dx := new(big.Int).Sub(p2.X, p1.X)
	dx.Mod(dx, P)

	dy := new(big.Int).Sub(p2.Y, p1.Y)
	dy.Mod(dy, P)

	inv := new(big.Int).ModInverse(dx, P)
	s := new(big.Int).Mul(dy, inv)
	s.Mod(s, P)

	s2 := new(big.Int).Mul(s, s)
	s2.Mod(s2, P)

	xr := new(big.Int).Sub(s2, new(big.Int).Add(p1.X, p2.X))
	xr.Mod(xr, P)

	yr := new(big.Int).Sub(p1.X, xr)
	yr.Mul(yr, s)
	yr.Sub(yr, p1.Y)
	yr.Mod(yr, P)

	return Affine{X: xr, Y: yr}
}

// double doubles a point
func double(p Affine) Affine {
	if isInfinity(p) || p.Y.Cmp(big.NewInt(0)) == 0 {
		return Affine{X: big.NewInt(0), Y: big.NewInt(0)}
	}

	three := big.NewInt(3)
	x2 := new(big.Int).Mul(p.X, p.X)
	x2.Mod(x2, P)

	s := new(big.Int).Mul(three, x2)
	s.Mod(s, P)

	twoY := new(big.Int).Mul(big.NewInt(2), p.Y)
	twoY.Mod(twoY, P)

	inv := new(big.Int).ModInverse(twoY, P)
	s.Mul(s, inv)
	s.Mod(s, P)

	s2 := new(big.Int).Mul(s, s)
	s2.Mod(s2, P)

	xr := new(big.Int).Sub(s2, new(big.Int).Mul(big.NewInt(2), p.X))
	xr.Mod(xr, P)

	yr := new(big.Int).Sub(p.X, xr)
	yr.Mul(yr, s)
	yr.Sub(yr, p.Y)
	yr.Mod(yr, P)

	return Affine{X: xr, Y: yr}
}

// mul multiplies a point by a scalar
func mul(p Affine, k *big.Int) Affine {
	result := Affine{X: big.NewInt(0), Y: big.NewInt(0)}
	addend := p

	for k.Cmp(big.NewInt(0)) > 0 {
		if k.Bit(0) == 1 {
			result = add(result, addend)
		}
		addend = double(addend)
		k.Rsh(k, 1)
	}

	return result
}

// liftXEvenY lifts x-coordinate to even-Y point
func liftXEvenY(x *big.Int) (Affine, error) {
	if x.Cmp(P) >= 0 {
		return Affine{}, ErrLiftXFailed
	}

	// y^2 = x^3 + 7 mod p
	c := new(big.Int).Mul(x, x)
	c.Mul(c, x)
	c.Add(c, B)
	c.Mod(c, P)

	// y = c^((p+1)/4) mod p
	y := new(big.Int).Exp(c, SQRT_EXP, P)

	// Verify y^2 == c
	y2 := new(big.Int).Mul(y, y)
	y2.Mod(y2, P)
	if y2.Cmp(c) != 0 {
		return Affine{}, ErrLiftXFailed
	}

	// Enforce even Y
	if y.Bit(0) == 1 {
		y.Sub(P, y)
	}

	return Affine{X: x, Y: y}, nil
}
