package service

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime/debug"
	"sync"
	"time"
	"workerChannel/helper"
)

const (
	LEVEL_DEBUG = "debug"
	LEVEL_INFO = "info"
	LEVEL_NOTICE = "notice"
	LEVEL_ERROR = "error"
	LEVEL_ALERT = "alert"
	LEVEL_CRITICAL = "critical"
)

var FilePoint *os.File
var FileName string
var LogLevel map[string]int8

type LogService struct {

}

var L *LogService

var chLog = make(chan bool, 1)
var LockPosition sync.Mutex

func GetLog() *LogService {
	chLog <- true
	if L == nil {
		L = &LogService{}
		LogLevel = map[string]int8{
			LEVEL_DEBUG:    1,
			LEVEL_INFO:     2,
			LEVEL_NOTICE:   3,
			LEVEL_ERROR:    4,
			LEVEL_ALERT:    5,
			LEVEL_CRITICAL: 6,
		}
	}
	<- chLog

	return L
}

func (Log *LogService)DebugOnError(err error) {
	if err != nil {
		L.Debug(err.Error(), LEVEL_DEBUG)
		L.outPut(fmt.Sprintf("%s\n", err))
	}
}

//打日志
func (Log *LogService) Debug(msg string, level string) {
	logLevel := int8(0)
	if _, ok := LogLevel[Cf.LogLevel]; ok {
		logLevel = LogLevel[Cf.LogLevel]
	}

	currentLevel := LogLevel[level]

	if currentLevel >= logLevel {
		dir := Cf.LogPath + helper.TimeFormat("/Ym", 0)

		//检查目录
		fileInfo, _ := os.Stat(dir)
		if fileInfo == nil || !fileInfo.IsDir() {
			err := os.Mkdir(dir, 0755)
			if err != nil {
				L.outPut(fmt.Sprintf("%s\n", err))
				return
			}
		}

		logFile := dir + helper.TimeFormat("/d_H", 0) + ".log"
		content := L.getMsg(msg, level)
		L.WriteLog(logFile, content)
		if currentLevel >= 4 {
			errorContent := string(debug.Stack())
			L.WriteLog(logFile, errorContent)
		}
	}
}

//日志信息格式化
func (Log *LogService) getMsg(msg string, level string) string {
	Nano := time.Now().Nanosecond() / 1000000
	msg = fmt.Sprintf("level:[%s]\ttime:[%s.%d]\tpid:[%d]\tmsg:[%s]", level, time.Now().Format("2006-01-02 15:04:05"), Nano, os.Getpid(), msg)
	L.outPut(msg)
	return msg + "\n"
}

func (Log *LogService)outPut(msg string) {
	fmt.Println(msg)
}

func (Log *LogService) WriteLog(fileName string, content string) {
	if FileName == "" || fileName != FileName {
		FileName = fileName
		FilePoint, _ = os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	}
	_, err := io.WriteString(FilePoint, content)
	if err != nil {
		L.outPut(fmt.Sprintf("%s", err.Error()))
	}
}

func (Log *LogService) WriteOverride(fileName string, content string) {
	LockPosition.Lock()
	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	defer func() {
		if err := file.Close(); err != nil {
			L.Debug("close file error"+err.Error(), LEVEL_ERROR)
		}
		LockPosition.Unlock()
	}()
	if err != nil {
		L.outPut(fmt.Sprintf("%s\n", err))
		return
	}
	_, err1 := io.WriteString(file, content)
	if err != nil {
		L.outPut(fmt.Sprintf("%s\n", err1))
	}
}

func (Log *LogService) ReadContent(fileName string) string {
	f, err := ioutil.ReadFile(fileName)
	if err != nil {
		L.Debug("文件读取失败:"+err.Error(), LEVEL_DEBUG)
		return ""
	}
	return string(f)
}