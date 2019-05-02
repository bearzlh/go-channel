package service

import (
	"encoding/json"
	"io/ioutil"
	"workerChannel/object"
)

func GetPosition(file string) object.Position {
	P := object.Position{}
	if Cf.Recover.From != "" {
		return P
	}
	content, err := ioutil.ReadFile(file)
	if err != nil {
		L.Debug("open position file err "+err.Error(), LEVEL_NOTICE)
		return P
	}
	errUnmarshal := json.Unmarshal(content, &P)
	if errUnmarshal != nil {
		L.Debug("position file unmarshal err "+errUnmarshal.Error(), LEVEL_NOTICE)
	}
	return P
}

func SetPosition(file string, P object.Position) {
	L.Debug("position save", LEVEL_DEBUG)
	content, err := json.Marshal(P)
	if err != nil {
		L.Debug("position file marshal err "+err.Error(), LEVEL_NOTICE)
	}
	L.WriteOverride(file, string(content))
}