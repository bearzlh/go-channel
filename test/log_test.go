package test

import (
	"testing"
	"workerChannel/service"
)

func TestFormat(t *testing.T) {
	service.L.Debug("aaa", service.LEVEL_ERROR)
}