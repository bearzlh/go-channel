package test

import (
	"os/exec"
	"strings"
	"testing"
	"workerChannel/helper"
	"workerChannel/service"
)

func TestCpu(t *testing.T) {
	shellPath := helper.GetPathJoin(service.Cf.AppPath, "host_info.sh cpu")
	out := exec.Command("/bin/bash","-c", shellPath)
	content, _ := out.Output()
	value := strings.TrimSpace(string(content))
	t.Log(value)
}
