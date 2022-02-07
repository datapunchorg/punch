package resource

import (
	"bytes"
	"io"
	"os/exec"
	"os"
)

func MinikubeExec(arg ...string) (string, error) {
	cmd := exec.Command("minikube", arg...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	err := cmd.Run()
	output := string(stdoutBuf.Bytes())
	if err != nil {
		output = string(stderrBuf.Bytes())
	}
	return output, err
}