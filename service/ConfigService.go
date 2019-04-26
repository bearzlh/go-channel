package service

import (
	"encoding/json"
	"github.com/fsnotify/fsnotify"
	"io/ioutil"
	"time"
	"workerChannel/helper"
)

const ConfigName = "config"
const Ext = ".json"
const Split = "/"

type ConfigService struct {
	ReadPath []ReadPath `json:"read_path"`
	Env      string     `json:"env"`
	AppId    string     `json:"appid"`
	Version  string     `json:"version"`
	Log      struct {
		Path       string `json:"path"`
		Level      string `json:"level"`
		FormatType string `json:"format_type"`
		Format     string `json:"format"`
	} `json:"log"`
	Factory struct {
		WorkerMax  int  `json:"worker_max"`
		WorkerInit int  `json:"worker_init"`
		On         bool `json:"on"`
	} `json:"factory"`
	Msg struct {
		IsBatch         bool   `json:"is_batch"`
		BatchSize       int    `json:"batch_size"`
		BatchTimeSecond int    `json:"batch_time_second"`
		SendType        string `json:"send_type"`
	} `json:"msg"`
	PhpTimeWindow int64    `json:"php_time_window"`
	AppPath       string `json:"app_path"`
	ServerPort    string `json:"server_port"`
	Es struct {
		Host           string `json:"host"`
		IndexFormat    string `json:"index_format"`
		Storage        string `json:"storage"`
		Retry          int    `json:"retry"`
		ConcurrentPost int    `json:"concurrent_post"`
	} `json:"es"`
	Monitor struct {
		Cpu        float64 `json:"cpu"`
		MemRestart float64 `json:"memory_restart"`
		MemStop    float64 `json:"memory_stop"`
		SleepIntervalNs int `json:"sleep_interval_ns"`
		PickInterval    int `json:"pick_interval"`
		CheckInterval   int `json:"check_interval"`
	} `json:"monitor"`
}

type ReadPath struct {
	Dir        string   `json:"dir"`
	TimeFormat string   `json:"time_format"`
	Suffix     string   `json:"suffix"`
	Type       string   `json:"type"`
	On         bool     `json:"on"`
	Continue   bool     `json:"continue"`
	Pick       string `json:"pick"`
}

var Cf *ConfigService

//保证单例模式
var ConfigBool = make(chan bool, 1)

//初始化配置，加载config配置文件
func GetConfig(path string) *ConfigService {
	ConfigBool <- true
	if Cf == nil {
		Cf = &ConfigService{}
		Cf.AppPath = path
		Cf.Log.Path = "log"
		Cf.loadFile()
	}
	<-ConfigBool
	return Cf
}

//加载配置文件
func (C *ConfigService) loadFile() *ConfigService {
	Cf.Log.Path = helper.GetPathJoin(Cf.AppPath, helper.GetPathWithoutSuffix(Cf.Log.Path))

	defaultFile := Cf.AppPath + Split + ConfigName + Ext

	content, err := ioutil.ReadFile(defaultFile)
	if err != nil {
		L.outPut("默认文件内容读取失败:" + defaultFile + ":" + err.Error())
		time.Sleep(time.Second)
		L.outPut("再次尝试")
		C.loadFile()
		return C
		//ExitProgramme(os.Interrupt)
	}

	err = json.Unmarshal(content, &Cf)

	if err != nil {
		L.outPut("默认内容解析错误:" + defaultFile + ":" + err.Error())
		time.Sleep(time.Second)
		L.outPut("再次尝试")
		C.loadFile()
		return C
		//ExitProgramme(os.Interrupt)
	}

	return C
}

func (C *ConfigService) ConfigWatch() {
	watch, err := fsnotify.NewWatcher()
	if err != nil {
		L.outPut(err.Error())
	}
	configFile := Cf.AppPath + Split + ConfigName + Ext
	err = watch.Add(configFile)
	L.outPut("watching config file:" + Cf.AppPath + Split + ConfigName + Ext)
	if err != nil {
		L.outPut(err.Error())
	}
	go func() {
		for {
			select {
			case ev := <-watch.Events:
				L.outPut(ev.Op.String())
				L.outPut("reload config file")
				workCount := Cf.Factory.WorkerInit
				buckStatus := Cf.Msg.IsBatch
				onStatus := Cf.Factory.On
				C.loadFile()
				err = watch.Add(configFile)
				if err != nil {
					L.outPut(err.Error())
				}

				if workCount != Cf.Factory.WorkerInit {
					if Cf.Factory.WorkerInit > Cf.Factory.WorkerMax {
						Cf.Factory.WorkerInit = Cf.Factory.WorkerMax
					}
					SetWorker(Cf.Factory.WorkerInit)
				}

				if onStatus != Cf.Factory.On {
					if Cf.Factory.On {
						StartWork()
					} else {
						StopWork()
					}
				}

				//切换批量发送状态
				if buckStatus != Cf.Msg.IsBatch {
					if Cf.Msg.IsBatch {
						Es.BuckWatch()
					} else {
						BuckClose <- true
					}
				}
			case err := <-watch.Errors:
				L.outPut(err.Error())
			}
		}
	}()
}
