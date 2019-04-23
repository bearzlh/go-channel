package service

import (
	"encoding/json"
	"io/ioutil"
	"workerChannel/object"
)

func GetPosition(file string) object.Position {
	content, err := ioutil.ReadFile(file)
	P := object.Position{}
	if err != nil {
		L.Debug("open position file err "+err.Error(), LEVEL_INFO)
		return P
	}
	errUnmarshal := json.Unmarshal(content, &P)
	if errUnmarshal != nil {
		L.Debug("position file unmarshal err "+errUnmarshal.Error(), LEVEL_INFO)
	}
	return P
}

func SetPosition(file string, P object.Position) {
	content, err := json.Marshal(P)
	if err != nil {
		L.Debug("position file marshal err "+err.Error(), LEVEL_INFO)
	}
	L.WriteOverride(file, string(content))
}

func SetRunTimePosition() {
	for _, rp := range Cf.ReadPath {
		if rp.On {
			file := GetPositionFile(rp.Type)
			tail := Tail[rp.Type]
			if tail != nil {
				var line int64
				if rp.Type == "php" {
					line = GetPhpPostLineNumber()
				} else {
					line = GetNginxPostLineNumber()
				}
				P := object.Position{File: tail.Filename, Line: line}
				SetPosition(file, P)
			}
		}
	}
}