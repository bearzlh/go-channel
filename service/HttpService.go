package service

import (
	"net/http"
	"os/exec"
	"workerChannel/helper"
)

//状态页打印
func Status(w http.ResponseWriter, req *http.Request) {
	_, err := w.Write(GetAnalysis(true))
	if err != nil {
		L.Debug(err.Error(), LEVEL_ERROR)
	}
}

//停止日志采集
func Stop(w http.ResponseWriter, req *http.Request) {
	StopCmd()
}

func StopCmd() {
	shellPath := helper.GetPathJoin(Cf.AppPath, "stop.sh stop")
	L.Debug("stop control:"+shellPath, LEVEL_ALERT)
	cmd := exec.Command("/bin/bash", "-c", shellPath)
	_, err := cmd.Output()
	if err != nil {
		L.Debug(err.Error(), LEVEL_ERROR)
	}
}

func Restart(w http.ResponseWriter, req *http.Request) {
	RestartCmd()
}

func RestartCmd() {
	shellPath := helper.GetPathJoin(Cf.AppPath, "stop.sh restart")
	L.Debug("stop control:"+shellPath, LEVEL_ALERT)
	cmd := exec.Command("/bin/bash", "-c", shellPath)
	_, err := cmd.Output()
	if err != nil {
		L.Debug(err.Error(), LEVEL_ERROR)
	}
}

//版本控制
func Update(w http.ResponseWriter, req *http.Request) {
	if req.FormValue("version") != "" {
		version := req.Form.Get("version")
		shellPath := helper.GetPathJoin(Cf.AppPath, "deploy.sh "+version)
		L.Debug("version control:"+shellPath, LEVEL_ALERT)
		cmd := exec.Command("/bin/bash", "-c", shellPath)
		_, err := cmd.Output()
		if err != nil {
			L.Debug(err.Error(), LEVEL_ERROR)
		}
	}
}
