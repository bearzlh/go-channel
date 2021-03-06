package test

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
	"workerChannel/helper"
	"workerChannel/service"
)

func TestIp(t *testing.T) {
	count := 0
	start := time.Now().UnixNano()
	t.Log(time.Now().UnixNano())
	for {
		count++
		service.IP.GetLocation("140.207.54.79")
		if count > 100000 {
			break
		}
	}
	t.Log((time.Now().UnixNano() - start) / 1000000000)
}

func TestGeo(t *testing.T) {
	t.Log(service.IP.GetLocation("10.250.0.186"))
}

func TestPostGeo(t *testing.T) {
	service.Es.Init()
	f, _ := os.Open("/Users/Bear/Desktop/doc.txt")
	rd := bufio.NewReader(f)
	for {
		line, _ := rd.ReadString('\n')
		line = strings.TrimSpace(line)
		guo, sheng, shi, jingwei := service.IP.GetLocation(line)
		if guo == "" && sheng == "" && shi == "" {
			continue
		}
		index := `{"index":{"_index":"geo","_type":"geo"}}`
		content := fmt.Sprintf(`{"guo":"%s","sheng":"%s","shi":"%s","jingwei":"%s"}`, guo, sheng, shi, jingwei)
		doc := index + "\n" + content + "\n"
		helper.FilePutContents(helper.GetPathJoin(service.Cf.AppPath, "storage/data"), doc, true)
		t.Log(helper.GetPathJoin(service.Cf.AppPath, "storage/data"), content)
		break
	}
	f.Close()
	service.Storage<-true
	select {

	}
}