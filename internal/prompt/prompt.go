package prompt

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Confirm asks a yes/no question. Default is No.
func Confirm(message string) (bool, error) {
	fmt.Fprintf(os.Stderr, "%s [y/N]: ", message)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes", nil
}

// Input asks for text input with an optional default value.
func Input(message, defaultVal string) (string, error) {
	if defaultVal != "" {
		fmt.Fprintf(os.Stderr, "%s [%s]: ", message, defaultVal)
	} else {
		fmt.Fprintf(os.Stderr, "%s: ", message)
	}
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return defaultVal, nil
	}
	return line, nil
}

// Password asks for a secret input with echo disabled (best-effort via stty).
func Password(message string) (string, error) {
	fmt.Fprintf(os.Stderr, "%s: ", message)

	// Disable terminal echo if possible.
	echoDisabled := disableEcho()
	if echoDisabled {
		defer func() {
			enableEcho()
			fmt.Fprintln(os.Stderr) // print newline after hidden input
		}()
	}

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// disableEcho turns off terminal echo using stty. Returns true on success.
func disableEcho() bool {
	cmd := exec.Command("stty", "-echo")
	cmd.Stdin = os.Stdin
	return cmd.Run() == nil
}

// enableEcho turns on terminal echo using stty.
func enableEcho() {
	cmd := exec.Command("stty", "echo")
	cmd.Stdin = os.Stdin
	_ = cmd.Run()
}
