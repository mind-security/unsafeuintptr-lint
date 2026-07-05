// Package somepkg provides an ordinary (non-keep-alive) function for the
// cross-package test cases.
package somepkg

// Ordinary does not keep its uintptr argument alive.
func Ordinary(uintptr) {}
