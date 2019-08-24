package test

import (
	"github.com/bitly/go-simplejson"
	"strings"
	"testing"
	"workerChannel/object"
)

func TestJson(t *testing.T) {
	type a struct {
		Name string `json:"name"`
	}
	b := make([]map[string]interface{}, 0)
	b = append(b, map[string]interface{}{"a":"b"})
	b = append(b, map[string]interface{}{"a":"b"})
	jsonTest := simplejson.New()
	jsonTest.SetPath([]string{"a", "b"}, "c")
	jsonTest.Set("a", b)
	t.Log(jsonTest)
}

func TestParams(t *testing.T) {
	str := `d2e950088ec16 [ info ] [ PARAM ] {"addtabs":"1"}`
	list := strings.Split(str, ` [ info ] [ PARAM ] `)
	if len(list) == 2 && strings.HasPrefix(list[1], "{") {
		param, _ := simplejson.NewJson([]byte(list[1]))
		a := object.Query{}
		m, _ := param.Map()
		for k,v:= range m{
			a.Key = k
			a.Value = v.(string)
		}
		t.Log(a)
	}
}