package test

import (
	"github.com/bitly/go-simplejson"
	"testing"
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
