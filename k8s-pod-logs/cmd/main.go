package main

import (
	"bufio"
	"fmt"
	"os"

	k8spodlogs "podlogs"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		// Prepend the current timestamp and "stdout F" to each line
		fmt.Printf("%s\n", k8spodlogs.ContainerdFromat(scanner.Text(), true))
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Error reading from stdin:", err)
	}
}
