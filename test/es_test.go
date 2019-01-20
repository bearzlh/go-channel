package test

import (
	"net/http"
	"testing"
	"workerChannel/object"
)

func TestTrimEmpty(t *testing.T) {
	var intMap map[string]*object.Query
	intMap = make(map[string]*object.Query, 100)
	printMemStats()
	intMap["0"] = &object.Query{Key: "aa", Value: "bb"}
	printMemStats()
	intMap["0"] = nil
	printMemStats()
	t.Log(len(intMap))
}

func TestEmpty(t *testing.T)  {
	_, err := http.Get("http://127.0.0.1:111")
	t.Log(err)
}