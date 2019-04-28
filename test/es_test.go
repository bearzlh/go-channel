package test

import (
	"encoding/json"
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

func TestResponse(t *testing.T) {
	msg := `{"errors":true,"items":[{"index":{"_id":"1","_index":"test","_type":"type","error":{"reason":"Rejecting mapping update to [test] as the final mapping would have more than 1 type: [a, type]","type":"illegal_argument_exception"},"status":400}},{"index":{"_id":"1","_index":"test","_type":"type","error":{"reason":"Rejecting mapping update to [test] as the final mapping would have more than 1 type: [a, type]","type":"illegal_argument_exception"},"status":400}},{"index":{"_id":"1DtpY2oBaeZTNpgaNIDW","_index":"test","_type":"type","error":{"reason":"Rejecting mapping update to [test] as the final mapping would have more than 1 type: [a, type]","type":"illegal_argument_exception"},"status":400}}],"took":24}`
	//msg := `{"errors":false,"items":[{"index":{"_id":"PTtrY2oBaeZTNpgaVoZN","_index":"test","_primary_term":1,"_seq_no":0,"_shards":{"failed":0,"successful":1,"total":2},"_type":"type","_version":1,"result":"created","status":201}},{"index":{"_id":"PjtrY2oBaeZTNpgaVoZN","_index":"test","_primary_term":1,"_seq_no":0,"_shards":{"failed":0,"successful":1,"total":2},"_type":"type","_version":1,"result":"created","status":201}}],"took":4}`
	res := new(object.EsBuckResponse)
	json.Unmarshal([]byte(msg), res)
	t.Log(res)
}