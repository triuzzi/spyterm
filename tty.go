package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"os/exec"
)

// findTTY walks up the process tree from the current PID to find the first
// ancestor with a real TTY (not "??"). This is necessary because Claude Code
// and other tools spawn subprocesses without a controlling terminal.
func findTTY() (string, error) {
	processID := os.Getpid()
	for range 10 {
		output, err := exec.Command("ps", "-o", "ppid=,tty=", "-p", strconv.Itoa(processID)).Output()
		if err != nil {
			break
		}
		fields := strings.Fields(strings.TrimSpace(string(output)))
		if len(fields) < 2 {
			break
		}
		tty := fields[1]
		parentProcessID, err := strconv.Atoi(fields[0])
		if err != nil {
			break
		}
		if tty != "??" && tty != "" {
			return "/dev/" + tty, nil
		}
		if parentProcessID <= 1 {
			break
		}
		processID = parentProcessID
	}
	return "", fmt.Errorf("could not detect TTY (walked process tree from PID %d)", os.Getpid())
}
