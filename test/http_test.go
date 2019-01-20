package test

import (
	"github.com/bitly/go-simplejson"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

func TestPost(t *testing.T) {
	res, _ := http.Post("http://192.168.0.109:9200/log/php", "application/json", strings.NewReader("{\"a\":1}"));
	t.Log(res.StatusCode)
	str, _ := ioutil.ReadAll(res.Body)
	result, _ := simplejson.NewJson([]byte(str))
	t.Log(result.Get("_shards").Get("successful"), string(str))
}