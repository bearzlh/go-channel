package test

import (
	"strings"
	"testing"
	"workerChannel/helper"
	"workerChannel/object"
	"workerChannel/service"
)

func TestSetPosition(t *testing.T) {
	for _, rp := range service.Cf.ReadPath {
		file := service.GetPositionFile(rp.Type)
		P := object.Position{File: service.GetLogFile(rp, 0), Line: 100}
		service.SetPosition(file, P)
	}
}

func TestGetPosition(t *testing.T) {
	for _, rp := range service.Cf.ReadPath {
		file := service.GetPositionFile(rp.Type)
		t.Log(service.GetPosition(file))
	}
}

func TestNextFile(t *testing.T) {
	timeLayout := service.Cf.ReadPath[0].TimeFormat
	layout := helper.FormatToLayout(timeLayout)
	file := "/data/www/cps/runtime/log/201903/08_13.log"
	format := strings.Replace(file, service.Cf.ReadPath[0].Dir, "", -1)
	format = strings.Replace(format, service.Cf.ReadPath[0].Suffix, "", -1)
	format = strings.Trim(format, "/")
	currentTime := helper.FormatTimeStamp(format, layout)
	t.Log(service.GetLogFile(service.Cf.ReadPath[0], currentTime + helper.GetMinDuration(timeLayout)))
	//time := helper.FormatTimeStamp(service.Cf.ReadPath[0].Dir+service.Cf.ReadPath[0].TimeFormat+service.Cf.ReadPath[0].Suffix)
}

func TestFormatToTpl(t *testing.T)  {
	positionObj := service.GetPosition(service.GetPositionFile("php"))
	currentLine := positionObj.Line
	position := service.GetPositionFromFileLine(positionObj.File, currentLine)
	t.Log(positionObj, position)
}

func TestGetNextFile(t *testing.T) {
	t.Log(service.GetNextFile(service.Cf.ReadPath[0], "/data/www/cps/runtime/log/201903/08_18.log"))
}