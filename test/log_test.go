package test

import (
	"testing"
	"workerChannel/service"
)

func TestFormat(t *testing.T) {
	t.Log(service.GetPositionFromFileLine("/usr/local/openresty/nginx/logs/access.log", 469611))
}