//go:build !windows

package main

// showFatal does nothing here: the error is printed on the console.
func showFatal(error) {}
