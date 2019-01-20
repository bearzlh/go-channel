package test

import (
	"regexp"
	"testing"
	"time"
	"workerChannel/helper"
	"workerChannel/object"
	"workerChannel/service"
)

func TestRegexp(t *testing.T) {
	bid := []byte("5c45b45c38751 [ info ] 获取缓存:PF:1 filecache命中")
	reg := regexp.MustCompile(`^[[:alnum:]]{13} `)
	s := reg.Find(bid)
	if string(s) == "" {
		t.Log(1)
	}else{
		t.Log(0)
	}
}

func TestLevel(t *testing.T) {
	wechatMatch := helper.RegexpMatch("wxasfds.aa.com", `^(wx\w+)\.`)
	t.Log(string(wechatMatch[0]), string(wechatMatch[1]))
}

func TestAppId(t *testing.T) {
	for  {
		service.L.Debug(time.Now().String(), "debug")
	}
}

func TestJson(t *testing.T) {
	nginxStr := `[100] 127.0.0.1 - - [09/Mar/2019:16:41:15 +0800] "www.dev.kpread.com:80" "GET /admin/auth/agent?ref=addtabs HTTP/1.1 status:302 cost:0.478 php:0.478 5" "http://www.dev.kpread.com/admin/templatemessage?ref=addtabs" "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.119 Safari/537.36" "-"127.0.0.1:9000`

	NginxMsg := object.NginxMsg{}
	service.ProcessNginxMsg(&NginxMsg, nginxStr)
	t.Log(NginxMsg)
}