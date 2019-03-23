package test

import (
	"github.com/bitly/go-simplejson"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"workerChannel/service"
)

func TestPost(t *testing.T) {
	res, _ := http.Post("http://192.168.0.109:9200/log/php", "application/json", strings.NewReader("{\"a\":1}"));
	t.Log(res.StatusCode)
	str, _ := ioutil.ReadAll(res.Body)
	result, _ := simplejson.NewJson([]byte(str))
	t.Log(result.Get("_shards").Get("successful"), string(str))
}

func TestUrl(t *testing.T) {
	urlStr := `https://wx4483982eb2874c7c.zsjwau.cn:443/t/6886704%!D(MISSING)]`
	u, _ := service.ParseUrl(urlStr)
	t.Log(u)
}