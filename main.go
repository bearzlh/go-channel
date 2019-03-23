package main

import (
	"fmt"
	"net/http"
	"os/exec"

	//_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
	"workerChannel/helper"
	"workerChannel/service"
)

func main() {
	SignalHandler()

	//获取当前路径
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		fmt.Println("error to get path" + err.Error())
	}

	//配置文件加载
	service.GetConfig(dir)

	//服务初始化
	ServiceInit()

	//启动web服务
	if service.Cf.ServerPort != "" {
		go func() {
			http.HandleFunc("/", Status)
			http.HandleFunc("/update", Update)
			service.L.Debug("status page,listen "+service.Cf.ServerPort, service.LEVEL_DEBUG)
			err := http.ListenAndServe(":"+service.Cf.ServerPort, nil)
			if err != nil {
				service.L.Debug(err.Error(), service.LEVEL_ERROR)
			}
		}()
	}

	//开始监控日志变化
	for i := 0; i < len(service.Cf.ReadPath); i++ {

		item := service.Cf.ReadPath[i]

		if item.On {
			//查看当前文件
			go func() {
				positionObj := service.GetPosition(service.GetPositionFile(item.Type))
				if positionObj.File != "" {
					service.TailNextFile(positionObj.File, item)
				} else {
					Log := service.GetLogFile(item, 0)
					service.TailNextFile(Log, item)
				}
			}()

			if item.TimeFormat != "" {
				//定时检查是不是需要切换文件
				go func(i int) {
					t := time.NewTimer(time.Second * 3)
					for {
						//以防配置文件修改后不生效
						item = service.Cf.ReadPath[i]
						select {
						case <-t.C:
							t.Reset(time.Second * 3)
							if service.Tail[item.Type] != nil && service.Tail[item.Type].Filename != "" {
								nextFile := service.GetNextFile(item, service.Tail[item.Type].Filename)
								if nextFile != "" {
									if time.Now().Unix()-service.An.TimeEnd > 2 {
										service.TailNextFile(nextFile, item)
									}
								}
							}
						}
					}
				}(i)
			}
		}
	}
	//errServe := http.ListenAndServe("127.0.0.1:6060", nil)
	//if errServe != nil {
	//	service.L.Debug("ListenAndServe error"+errServe.Error(), service.LEVEL_ERROR)
	//}
	select {}
}

//服务初始化
func ServiceInit() {
	//初始化日志
	service.GetLog()

	//初始化统计信息
	service.SetAnalysis()

	//检测es是否可用
	service.Es.CheckEsCanAccess()

	//工作初始化
	service.InitWorkPool()

	//配置文件监控
	service.Cf.ConfigWatch()

	//检测批量发送队列
	if service.Cf.Msg.IsBatch {
		service.Es.BuckWatch()
	}
}

//状态页打印
func Status(w http.ResponseWriter, req *http.Request) {
	_, err := w.Write(service.GetAnalysis(true))
	if err != nil {
		service.L.Debug(err.Error(), service.LEVEL_ERROR)
	}
}

//版本控制
func Update(w http.ResponseWriter, req *http.Request) {
	if req.FormValue("version") != "" {
		version := req.Form.Get("version")
		shellPath := helper.GetPathJoin(service.Cf.AppPath, "deploy.sh "+version)
		service.L.Debug("version control:"+shellPath, service.LEVEL_ALERT)
		cmd := exec.Command("/bin/bash", "-c", shellPath)
		_, err := cmd.Output()
		if err != nil {
			service.L.Debug(err.Error(), service.LEVEL_ERROR)
		}
	}
}

//监听并处理信号，保存状态信息
func SignalHandler() {
	fmt.Println("signal watching!")
	signal.Notify(service.StopSignal, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
	go func() {
		msg := <-service.StopSignal //阻塞等待
		//保存当前进度
		service.L.Debug(msg.String(), service.LEVEL_DEBUG)
		service.SetRunTimePosition()
		service.L.WriteOverride(helper.GetPathJoin(service.Cf.AppPath, ".analysis"), string(service.GetAnalysis(false)))
		os.Exit(0)
	}()
}
