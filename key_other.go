//go:build !windows

package main

import (
	"bufio"
	"fmt"
	"os"
)

func waitForKey() error {
	fmt.Print("Press Enter to start processing...")
	if _, err := bufio.NewReader(os.Stdin).ReadString('\n'); err != nil {
		return err
	}
	return nil
}
