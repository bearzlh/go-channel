package test

import (
	"path/filepath"
	"testing"
	"workerChannel/service"
)

func TestMain(m *testing.M) {
	dir, _ := filepath.Abs("..")
	service.GetConfig(dir)
	service.GetLog(service.Cf)
	m.Run()
}