// Code generated by wit-bindgen-go. DO NOT EDIT.

//go:build !wasip1

// Package insecure represents the imported interface "wasi:random/insecure@0.2.0".
//
// The insecure interface for insecure pseudo-random numbers.
//
// It is intended to be portable at least between Unix-family platforms and
// Windows.
package insecure

import (
	"internal/cm"
)

// GetInsecureRandomBytes represents the imported function "get-insecure-random-bytes".
//
// Return `len` insecure pseudo-random bytes.
//
// This function is not cryptographically secure. Do not use it for
// anything related to security.
//
// There are no requirements on the values of the returned bytes, however
// implementations are encouraged to return evenly distributed values with
// a long period.
//
//	get-insecure-random-bytes: func(len: u64) -> list<u8>
//
//go:nosplit
func GetInsecureRandomBytes(len_ uint64) (result cm.List[uint8]) {
	len0 := (uint64)(len_)
	wasmimport_GetInsecureRandomBytes((uint64)(len0), &result)
	return
}

//go:wasmimport wasi:random/insecure@0.2.0 get-insecure-random-bytes
//go:noescape
func wasmimport_GetInsecureRandomBytes(len0 uint64, result *cm.List[uint8])

// GetInsecureRandomU64 represents the imported function "get-insecure-random-u64".
//
// Return an insecure pseudo-random `u64` value.
//
// This function returns the same type of pseudo-random data as
// `get-insecure-random-bytes`, represented as a `u64`.
//
//	get-insecure-random-u64: func() -> u64
//
//go:nosplit
func GetInsecureRandomU64() (result uint64) {
	result0 := wasmimport_GetInsecureRandomU64()
	result = (uint64)((uint64)(result0))
	return
}

//go:wasmimport wasi:random/insecure@0.2.0 get-insecure-random-u64
//go:noescape
func wasmimport_GetInsecureRandomU64() (result0 uint64)
