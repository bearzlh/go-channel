package test

import (
	"fmt"
	"github.com/hpcloud/tail"
	"testing"
)

func TestTail(t *testing.T) {
	filePath := "/Users/Bear/gopath/src/workerChannel/test/config_test.go"
	tailFile, _ := tail.TailFile(filePath, tail.Config{Follow: false})
	for lines := range tailFile.Lines {
		pos ,_ := tailFile.Tell()
		t.Log(lines.Text+fmt.Sprintf("%d", pos))
	}
}