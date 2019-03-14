package test

import (
	"fmt"
	"github.com/hpcloud/tail"
	"testing"
	"time"
)

func TestTail(t *testing.T) {
	hourLine := 200000 * 75
	rate := 3600000000 / hourLine / 4
	filePath := "/var/log/cpslog/201903/21_10.log"
	tailFile, _ := tail.TailFile(filePath, tail.Config{Follow: false})
	now := time.Now()
	for range tailFile.Lines {
		time.Sleep(time.Duration(rate) * time.Microsecond)
		//fmt.Println(line.Text)
	}
	fmt.Println(time.Now().Sub(now))
}