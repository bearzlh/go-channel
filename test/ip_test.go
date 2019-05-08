package test

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
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
		service.GetLocation("140.207.54.79")
		if count > 100000 {
			break
		}
	}
	t.Log((time.Now().UnixNano() - start) / 1000000000)
}

func TestPostGeo(t *testing.T) {
	service.Es.Init()
	service.Lock = new(sync.Mutex)
	f, _ := os.Open("/Users/Bear/Desktop/doc.txt")
	rd := bufio.NewReader(f)
	for {
		line, _ := rd.ReadString('\n')
		line = strings.TrimSpace(line)
		zhou, guo, sheng, shi, jingwei := service.GetLocation(line)
		if guo == "" && sheng == "" && shi == "" {
			continue
		}
		index := `{"index":{"_index":"geo","_type":"geo"}}`
		content := fmt.Sprintf(`{"zhou":"%s","guo":"%s","sheng":"%s","shi":"%s","jingwei":"%s"}`, zhou, guo, sheng, shi, jingwei)
		doc := index + "\n" + content + "\n"
		helper.FilePutContents(helper.GetPathJoin(service.Cf.AppPath, "storage/data"), doc, true)
		t.Log(helper.GetPathJoin(service.Cf.AppPath, "storage/data"), content)
	}
	f.Close()
	service.Storage<-true
	select {

	}
}