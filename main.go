package main

import (
	"fmt"
	"github.com/hpcloud/tail"
	"net/http"
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

	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	service.GetLog()
	service.GetConfig(dir)
	service.SetAnalysis()
	service.Es.CheckEsCanAccess()
	err := service.InitWorkPool()
	if err != nil {
		fmt.Printf(err.Error())
		os.Exit(1)
	}
	go func() {
		service.Dispatch()
	}()

	//启动web服务
	go func() {
		http.HandleFunc("/", Status)
		err := http.ListenAndServe(":"+service.Cf.Port, nil)
		if err != nil {
			service.L.Debug(err.Error(), service.LEVEL_ERROR)
		}
	}()

	service.LineMap = make(map[string]service.LineItem)

	//配置文件监控
	service.Cf.ConfigWatch()

	//批量发送
	if service.Cf.Es.BuckPost {
		service.Es.BuckWatch()
	}

	service.Tail = make(map[string]*tail.Tail)
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
				go func() {
					for {
						timer := time.NewTimer(time.Second * 5)
						select {
						case <-timer.C:
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
				}()
			}
		}
	}

	select {}
}

//状态页打印
func Status(w http.ResponseWriter, req *http.Request) {
	_, err := w.Write(service.GetAnalysis())
	if err != nil {
		service.L.Debug(err.Error(), service.LEVEL_ERROR)
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
		service.L.WriteOverride(helper.GetPathJoin(service.Cf.AppPath, ".analysis"), string(service.GetAnalysis()))
		os.Exit(0)
	}()
}