//go:build windows

package main

import (
	"fmt"
	"os"
	"syscall"
)

func waitForKey() error {
	fmt.Print("Press any key to start processing...")

	console, err := os.OpenFile("CONIN$", os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("open console input: %w", err)
	}
	defer console.Close()

	handle := syscall.Handle(console.Fd())
	var originalMode uint32
	if err := syscall.GetConsoleMode(handle, &originalMode); err != nil {
		return fmt.Errorf("read console mode: %w", err)
	}

	const enableLineInput = 0x0002
	const enableEchoInput = 0x0004
	setConsoleMode := syscall.NewLazyDLL("kernel32.dll").NewProc("SetConsoleMode")
	mode := originalMode &^ (enableLineInput | enableEchoInput)
	result, _, callErr := setConsoleMode.Call(uintptr(handle), uintptr(mode))
	if result == 0 {
		return fmt.Errorf("set console mode: %w", callErr)
	}
	defer setConsoleMode.Call(uintptr(handle), uintptr(originalMode))

	var key uint16
	var read uint32
	if err := syscall.ReadConsole(handle, &key, 1, &read, nil); err != nil {
		return fmt.Errorf("read confirmation key: %w", err)
	}
	fmt.Println()
	return nil
}
