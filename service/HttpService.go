package service

import (
	"mq/service"
	"net/http"
	"os/exec"
	"workerChannel/helper"
)

//启动http服务
func StartHttp() {
	//启动web服务
	if Cf.ServerPort != "" {
		go func() {
			http.HandleFunc("/", Status)
			http.HandleFunc("/config", Config)
			http.HandleFunc("/update", Update)
			http.HandleFunc("/stop", Stop)
			http.HandleFunc("/restart", Restart)
			L.Debug("status page,listen "+Cf.ServerPort, LEVEL_INFO)
			err := http.ListenAndServe(Cf.ServerPort, nil)
			if err != nil {
				service.L.Debug(err.Error(), service.LEVEL_ERROR)
			}
		}()
	}
}

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
	shellPath := helper.GetPathJoin(Cf.AppPath, "controller.sh stop")
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

func Config(w http.ResponseWriter, req *http.Request) {
	if req.FormValue("key") != "" && req.FormValue("value") != ""{
		key := req.Form.Get("key")
		value := req.Form.Get("value")
		shellPath := helper.GetPathJoin(Cf.AppPath, "controller.sh update_config "+key+" "+value)
		L.Debug("update config:"+shellPath, LEVEL_ALERT)
		cmd := exec.Command("/bin/bash", "-c", shellPath)
		content, err := cmd.Output()
		if err != nil {
			L.Debug(err.Error(), LEVEL_ERROR)
		}
		L.Debug(string(content), LEVEL_INFO)
	}
}

func RestartCmd() {
	shellPath := helper.GetPathJoin(Cf.AppPath, "controller.sh restart")
	L.Debug("restart control:"+shellPath, LEVEL_ALERT)
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
