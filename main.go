package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func main() {
	if err := RunElevated("sysctl", "net.ipv4.ip_forward=1"); err != nil {
		fmt.Println("error running sysctl")
	}
}

// Run runs binary `bin` with `args`.
func Run(bin string, args ...string) error {
	return run(bin, os.Stdout, false, args...)
}

// RunElevated runs binary with escalation (admin / root)
func RunElevated(bin string, args ...string) error {
	return run(bin, os.Stdout, true, args...)
}

// RunElevatedWithResultReader runs binary `bin` with `args` returning stdout contents as `io.Reader`
func RunElevatedWithResultReader(bin string, args ...string) (io.Reader, error) {
	stdout := bytes.NewBuffer(nil)

	return stdout, run(bin, stdout, true, args...)
}

// RunElevatedWithResult runs binary `bin` with `args` returning stdout contents as bytes.
func RunElevatedWithResult(bin string, args ...string) ([]byte, error) {
	stdout, err := RunElevatedWithResultReader(bin, args...)
	if err != nil {
		return nil, err
	}
	stdoutBytes, err := ioutil.ReadAll(stdout)
	if err != nil {
		return nil, fmt.Errorf("failed to read stdout: %w", err)
	}

	return stdoutBytes, nil
}

// RunWithResultReader runs binary `bin` with `args` returning stdout contents as `io.Reader`.
func RunWithResultReader(bin string, args ...string) (io.Reader, error) {
	stdout := bytes.NewBuffer(nil)

	return stdout, run(bin, stdout, false, args...)
}

// RunWithResult runs binary `bin` with `args` returning stdout contents as bytes.
func RunWithResult(bin string, args ...string) ([]byte, error) {
	stdout, err := RunWithResultReader(bin, args...)
	if err != nil {
		return nil, err
	}

	stdoutBytes, err := ioutil.ReadAll(stdout)
	if err != nil {
		return nil, fmt.Errorf("failed to read stdout: %w", err)
	}

	return stdoutBytes, nil
}

func run(bin string, stdout io.Writer, withEscalate bool, args ...string) error {
	binArgs := []string{bin}
	binArgs = append(binArgs, args...)

	var fullCmd string
	var cmd *exec.Cmd
	if withEscalate {
		switch runtime.GOOS {
		case "linux":
			fullCmd = "pkexec" + " " + bin + " " + strings.Join(args, " ")
			cmd = exec.Command("pkexec", binArgs...) //nolint:gosec
		case "windows":
			fullCmd = "runas" + " " + bin + " " + strings.Join(args, " ")
			cmd = exec.Command("runas", binArgs...) //nolint:gosec
		default:
			fullCmd = "sudo" + " " + bin + " " + strings.Join(args, " ")
			cmd = exec.Command("sudo", binArgs...) //nolint:gosec
		}
	} else {
		fullCmd = bin + " " + strings.Join(args, " ")
		cmd = exec.Command(bin, args...)
	}

	stderrBuf := bytes.NewBuffer(nil)

	cmd.Stderr = io.MultiWriter(os.Stderr, stderrBuf)
	cmd.Stdout = stdout
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return NewErrorWithStderr(fmt.Errorf("error running command \"%s\": %w", fullCmd, err),
			stderrBuf.Bytes())
	}

	return nil
}

// ErrorWithStderr is an error raised by the external process.
// `Err` is an actual error coming from `exec`, while `Stderr` contains
// stderr output of the process.
type ErrorWithStderr struct {
	Err    error
	Stderr []byte
}

// NewErrorWithStderr constructs new `ErrorWithStderr`.
func NewErrorWithStderr(err error, stderr []byte) *ErrorWithStderr {
	return &ErrorWithStderr{
		Err:    err,
		Stderr: stderr,
	}
}

// Error implements `error`.
func (e *ErrorWithStderr) Error() string {
	return e.Err.Error() + ": " + string(e.Stderr)
}
