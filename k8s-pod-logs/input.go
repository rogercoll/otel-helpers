package k8spodlogs

import (
	"fmt"
	"time"
)

func ContainerdFromat(intput string, stdout bool) string {
	// Get the current timestamp
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000000000Z")
	output := "stdout"
	if !stdout {
		output = "stderr"
	}

	return fmt.Sprintf("%s %s", fmt.Sprintf("%s %s F", timestamp, output), intput)
}
