package test

import (
	"github.com/gofrs/uuid"
	"log"
	"runtime"
	"strings"
	"testing"
	"time"
	"workerChannel/helper"
	"workerChannel/service"
)

func TestConfig(t *testing.T) {
	MapList := make(map[string]string)
	printMemStats()
	for i := 0; i < 1000000; i++ {
		uuidStr, _ := uuid.NewV4()
		MapList[uuidStr.String()] = uuidStr.String()
	}
	printMemStats()
	for index, _ := range MapList {
		delete(MapList, index)
	}
	runtime.GC()
	MapList = make(map[string]string, 0)
	printMemStats()
	select{}
}

func printMemStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	log.Printf("Alloc = %v TotalAlloc = %v Sys = %v NumGC = %v\n", m.Alloc/1024, m.TotalAlloc/1024, m.Sys/1024, m.NumGC)
}

func TestTime(t *testing.T) {
	//t1 := time.Date(2018, 2, 1, 1, 0, 2, 0, time.Local);
	//t2 := time.Date(2018, 2, 2, 0, 1, 1, 0, time.Local);
	//t3 := time.Duration(t2.UnixNano() - t1.UnixNano())
	//t.Log(t3.String())

	t.Log(time.Now().Sub(time.Now()))
}

func TestBranch(t *testing.T) {
	out := helper.ExecShellPipe([]string{"git", "branch", "-a"}, []string{"grep", "-i", "Master"})
	t.Log("aaa"+out+"bbb")
}

func TestAn(t *testing.T) {
 	t.Log(strings.Contains(service.Cf.ReadPath[0].Pick, "cooki1e"))
}

func TestHost(t *testing.T) {
	t.Log(service.Es.GetHost())
}