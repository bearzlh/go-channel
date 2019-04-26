package test

import (
	"bytes"
	"fmt"
	"testing"
	"text/template"
	"time"
	"workerChannel/helper"
	"workerChannel/object"
	"workerChannel/service"
)

func TestLine(t *testing.T) {
	t.Run("test shell", func(t *testing.T) {
		cmdOut := service.GetFileEndLine("/Users/Bear/gopath/src/workerChannel/test/work_test.go")
		shellOut := helper.ExecShellPipe([]string{"wc", "/Users/Bear/gopath/src/workerChannel/test/work_test.go"}, []string{"awk", "{print $1}"})
		if fmt.Sprintf("%d", cmdOut) != shellOut {
			t.Fail()
		}
	})
}

func TestTpl(t *testing.T) {
	type b struct {
		Name string
		Year int
	}
	boy := b{"zhenglh", 10}
	var FuncMap = template.FuncMap{
		"year": year,
	}
	//const (
	//	master  = `Names:{{block "list" .}}{{"\n"}}{{range .}}{{println "-" .}}{{end}}{{end}}`
	//	overlay = `{{define "list"}} {{join . ", "}}{{end}} `
	//)
	mul := "2"
	tpl, _ := template.New("test").Funcs(FuncMap).Parse(`Name-{{.Name}} Year-{{block "t" .}}{{end}}`)
	year, _ := template.Must(tpl.Clone()).Parse(`{{define "t"}}{{year .Year `+mul+`}}{{end}}`)
	var re bytes.Buffer
	err := year.Execute(&re, boy)
	if err != nil {
		t.Log(err.Error())
	}
	t.Log(re.String())
}

func year(year int, mul int) int {
	return year * mul
}

func TestPad(t *testing.T) {
	doc := object.Doc{}
	doc.Index = object.Index{IndexName: object.IndexContent{Index: "a", Type: "b"}}
	doc.Content = object.PhpMsg{}
	service.BuckDoc<-doc
	t.Log(<-service.BuckDoc)
}

func TestRun(t *testing.T) {
	jobMap := make(map[int64][]string)
	jobMap[1] = []string{"a","b","c"}
	jobMap[4] = []string{"a","b","c"}
	jobMap[3] = []string{"a","b","c"}
	jobMap[5] = []string{"a","b","c"}
	var keys []int64
	for k := range jobMap {
		keys = append(keys, k)
	}
	for _, k := range keys {
		if time.Now().Unix() - k > 1 {
			for _, job := range jobMap[k] {
				service.JobQueue<-job
			}
		}
	}
}

func TestSub(t *testing.T) {
	now := time.Now()
	time.Sleep(time.Millisecond * 10)
	t.Log(float64(time.Now().Sub(now)))
}

func TestWorker(t *testing.T)  {
	t.Log(object.GetIndex("", service.Cf.Es.IndexFormat, time.Now().Unix(), "php"))
}