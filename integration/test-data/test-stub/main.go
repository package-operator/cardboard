package main

import (
	"fmt"
	"os"
	"os/signal"
)

// Test Stub to use as workload stand-in in integration tests.
func main() {
	fmt.Fprintln(os.Stdout, "waiting for something interesting to happen...")

	// block forever
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)

	<-sigs
	fmt.Fprintln(os.Stdout, "shutdown...")
}
