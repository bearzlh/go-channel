package service

import (
	"encoding/json"
	"io/ioutil"
	"workerChannel/helper"
	"workerChannel/object"
)

func GetPosition(file string) object.Position {
	P := object.Position{}
	content, err := ioutil.ReadFile(file)
	if err != nil {
		L.Debug("open position file err "+err.Error(), LEVEL_NOTICE)
		return P
	}

	if len(content) > 0 {
		errUnmarshal := json.Unmarshal(content, &P)
		if errUnmarshal != nil {
			L.Debug("position file unmarshal err "+errUnmarshal.Error(), LEVEL_NOTICE)
		}
	}

	return P
}

func SetPosition(file string, P object.Position) {
	L.Debug("position saving", LEVEL_INFO)
	content, err := json.Marshal(P)
	if err != nil {
		L.Debug("position file marshal err "+err.Error(), LEVEL_ERROR)
	}
	errWrite := helper.FilePutContents(file, string(content), false)
	if errWrite != nil {
		L.Debug("position file write err "+err.Error(), LEVEL_ERROR)
	}
}