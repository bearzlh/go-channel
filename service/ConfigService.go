package service

import (
	"encoding/json"
	"github.com/fsnotify/fsnotify"
	"io/ioutil"
	"os"
	"workerChannel/helper"
)

const ConfigName = "config"
const Ext = ".json"
const Split = "/"

type ConfigService struct {
	Port         string `json:"port"`
	ReadPath     []ReadPath `json:"read_path"`
	LogPath      string `json:"log_path"`
	LogLevel     string `json:"log_level"`
	PositionFile string `json:"position_file"`
	ImportFile   string `json:"import_file"`
	Time         int    `json:"time"`
	WorkerMax    int    `json:"worker_max"`
	WorkerTotal  int    `json:"worker_total"`
	JobForWork   int    `json:"job_for_work"`
	AppPath      string `json:"app_path"`
	Es           struct {
		Host      string `json:"host"`
		BuckSize  int    `json:"buck_size"`
		BuckPost  bool   `json:"buck_post"`
	} `json:"es"`
	CollectAnalysis bool `json:"collect_analysis"`
}

type ReadPath struct {
	Dir        string `json:"dir"`
	TimeFormat string `json:"time_format"`
	Suffix     string `json:"suffix"`
	Type       string `json:"type"`
	On         bool   `json:"on"`
	Continue   bool   `json:"continue"`
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
		Cf.LogPath = "log"
		Cf.loadFile()
	}
	<-ConfigBool
	return Cf
}

//加载配置文件
func (C *ConfigService) loadFile() *ConfigService {
	Cf.LogPath = helper.GetPathJoin(Cf.AppPath, helper.GetPathWithoutSuffix(Cf.LogPath))

	defaultFile := Cf.AppPath + Split + ConfigName + Ext

	content, err := ioutil.ReadFile(defaultFile)
	if err != nil {
		L.Debug("默认文件内容读取失败:"+defaultFile+":"+err.Error(), LEVEL_ERROR)
		ExitProgramme(os.Interrupt)
	}

	err = json.Unmarshal(content, &Cf)

	if err != nil {
		L.Debug("默认内容解析错误:"+defaultFile+":"+err.Error(), LEVEL_ERROR)
		ExitProgramme(os.Interrupt)
	}

	return C
}

func (C *ConfigService) ConfigWatch() {
	watch,err := fsnotify.NewWatcher()
	if err != nil {
		L.Debug(err.Error(), LEVEL_ERROR)
	}
	configFile := Cf.AppPath + Split + ConfigName + Ext
	err = watch.Add(configFile)
	L.Debug("watching config file:"+Cf.AppPath + Split + ConfigName + Ext, LEVEL_DEBUG)
	if err != nil {
		L.Debug(err.Error(), LEVEL_ERROR)
	}
	go func() {
		for {
			select {
			case ev := <-watch.Events:
				L.Debug(ev.Op.String(), LEVEL_DEBUG)
				L.Debug("reload config file", LEVEL_DEBUG)
				workCount := Cf.WorkerTotal
				buckStatus := Cf.Es.BuckPost
				C.loadFile()
				err = watch.Add(configFile)
				if err != nil {
					L.Debug(err.Error(), LEVEL_ERROR)
				}
				if workCount < Cf.WorkerTotal {
					if Cf.WorkerTotal > Cf.WorkerMax {
						Cf.WorkerTotal = Cf.WorkerMax
					}
					AddWorker(Cf.WorkerTotal - workCount)
				}
				//切换批量发送状态
				if buckStatus != Cf.Es.BuckPost {
					if Cf.Es.BuckPost {
						Es.BuckWatch()
					} else {
						BuckClose<-true
					}
				}
			case err := <-watch.Errors:
				L.Debug(err.Error(), LEVEL_ERROR)
			}
		}
	}()
}