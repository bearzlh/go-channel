package helper

import (
	"bufio"
	"bytes"
	"os/exec"
	"strings"
)

func ExecShellPipe(p1 []string, p2 []string) string {
	cmd1 := exec.Command(p1[0], p1[1:]...)
	cmd2 := exec.Command(p2[0], p2[1:]...)
	stdout1, err := cmd1.StdoutPipe()
	if err != nil {
		return ""
	}
	if err := cmd1.Start(); err != nil {
		return ""
	}
	outputBuf1 := bufio.NewReader(stdout1)
	stdin2, err := cmd2.StdinPipe()
	if err != nil {
		return ""
	}
	_, err1 := outputBuf1.WriteTo(stdin2)
	if err1 != nil {
		return ""
	}

	var outputBuf2 bytes.Buffer
	cmd2.Stdout = &outputBuf2
	if err := cmd2.Start(); err != nil {
		return ""
	}
	err = stdin2.Close()
	if err != nil {
		return ""
	}
	if err := cmd2.Wait(); err != nil {
		return ""
	}

	return strings.Trim(outputBuf2.String(), "\n")
}