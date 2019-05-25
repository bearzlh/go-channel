package service

import (
	"encoding/json"
	"github.com/fsnotify/fsnotify"
	"io/ioutil"
	"strings"
	"time"
	"workerChannel/helper"
	"workerChannel/object"
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
		Path        string `json:"path"`
		Level       string `json:"level"`
		FormatTime  string `json:"format_time"`
		FormatLevel bool   `json:"format_level"`
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
		IpCacheTime     int64  `json:"ip_cache_time"`
		IpCheckInterval int64  `json:"ip_check_interval"`
	} `json:"msg"`
	PhpTimeWindow int64  `json:"php_time_window"`
	AppPath       string `json:"app_path"`
	ServerPort    string `json:"server_port"`
	Es            struct {
		Host           string `json:"host"`
		IndexFormat    string `json:"index_format"`
		Storage        string `json:"storage"`
		Retry          int    `json:"retry"`
		ConcurrentPost int    `json:"concurrent_post"`
		RecoverThread  int    `json:"recover_thread"`
	} `json:"es"`
	Monitor struct {
		Cpu             float64 `json:"cpu"`
		Load            float64 `json:"load"`
		MemRestart      float64 `json:"memory_restart"`
		MemStop         float64 `json:"memory_stop"`
		SleepIntervalNs int     `json:"sleep_interval_ns"`
		SleepTimeSet    float64 `json:"sleep_time_set"`
		PickInterval    int     `json:"pick_interval"`
		CheckInterval   int     `json:"check_interval"`
	} `json:"monitor"`
	Recover struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"recover"`
}

type ReadPath struct {
	Dir        string `json:"dir"`
	TimeFormat string `json:"time_format"`
	Suffix     string `json:"suffix"`
	Type       string `json:"type"`
	On         bool   `json:"on"`
	Continue   bool   `json:"continue"`
	Pick       string `json:"pick"`
	AppId      string `json:"appid"`
}

//全局
var Cf *ConfigService

//初始化配置，加载config配置文件
func GetConfig(path string) *ConfigService {
	Cf = &ConfigService{}
	Cf.AppPath = path
	Cf.loadFile()
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
	}

	err = json.Unmarshal(content, &Cf)

	if err != nil {
		L.outPut("默认内容解析错误:" + defaultFile + ":" + err.Error())
		time.Sleep(time.Second)
		L.outPut("再次尝试")
		C.loadFile()
		return C
	}

	return C
}

//配置文件动态加载
func (C *ConfigService) ConfigWatch() {
	watch, err := fsnotify.NewWatcher()
	if err != nil {
		L.outPut(err.Error())
	}
	configFile := helper.GetPathJoin(Cf.AppPath, ConfigName+Ext)
	err = watch.Add(configFile)
	L.outPut("watching config file:" + configFile)
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
				postCount := Cf.Es.ConcurrentPost
				buckStatus := Cf.Msg.IsBatch
				onStatus := Cf.Factory.On
				C.loadFile()
				err = watch.Add(configFile)
				if err != nil {
					L.outPut(err.Error())
				}

				if Cf.Monitor.SleepTimeSet != 0 {
					object.SleepTime = Cf.Monitor.SleepTimeSet
				}

				if workCount != Cf.Factory.WorkerInit {
					if Cf.Factory.WorkerInit > Cf.Factory.WorkerMax {
						Cf.Factory.WorkerInit = Cf.Factory.WorkerMax
					}
					SetWorker(Cf.Factory.WorkerInit)
				}

				if postCount != Cf.Es.ConcurrentPost {
					ConcurrentPost = make(chan int, Cf.Es.ConcurrentPost)
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

//日志等级配置
func (C *ConfigService) GetLevel() string {
	return C.Log.Level
}

//日志格式参数
func (C *ConfigService) GetFormatTime() string {
	return C.Log.FormatTime
}

//日志格式类型
func (C *ConfigService) GetFormatLevel() bool {
	return C.Log.FormatLevel
}

//日志路径
func (C *ConfigService) GetPath() string {
	if !strings.HasPrefix(Cf.Log.Path, "/") {
		Cf.Log.Path = helper.GetPathJoin(Cf.AppPath, Cf.Log.Path)
	}
	return C.Log.Path
}
