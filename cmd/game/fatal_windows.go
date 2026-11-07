package main

import (
	"syscall"
	"unsafe"
)

// showFatal shows an error in a message box: a release build has no console to print it on.
func showFatal(err error) {
	text, _ := syscall.UTF16PtrFromString(err.Error())
	title, _ := syscall.UTF16PtrFromString("Oldworld2")
	const iconError = 0x10
	syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), iconError)
}
