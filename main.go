package main

import (
	"fmt"
	//_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
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
	service.Es.Init()

	//工作初始化
	service.InitWorkPool()

	//配置文件监控
	service.Cf.ConfigWatch()

	//启动http服务
	service.StartHttp()

	//启动日志监控
	service.StartWork()
}

//监听并处理信号，保存状态信息
func SignalHandler() {
	signal.Notify(service.StopSignal, os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
	go func() {
		msg := <-service.StopSignal //阻塞等待
		//保存当前状态
		service.L.Debug("信息中断，信号信息"+msg.String(), service.LEVEL_CRITICAL)
		service.StopWork()
		os.Exit(0)
	}()
}
