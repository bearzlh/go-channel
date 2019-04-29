package service

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"runtime/debug"
	"strings"
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
		L.setLogDir()
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
	if _, ok := LogLevel[Cf.Log.Level]; ok {
		logLevel = LogLevel[Cf.Log.Level]
	}

	currentLevel := LogLevel[level]

	if currentLevel >= logLevel {
		logFile := L.getLogPath(level)
		err := helper.Mkdir(path.Dir(logFile))
		if err != nil {
			L.outPut(fmt.Sprintf("%s\n", err))
			ExitProgramme(os.Interrupt)
		}
		content := L.getMsg(msg, level)
		L.WriteLog(logFile, content)
		if currentLevel >= 4 {
			errorContent := string(debug.Stack())
			L.WriteLog(logFile, errorContent)
		}
		switch level {
		case LEVEL_ERROR:
			Lock.Lock()
			An.CodeError++
			Lock.Unlock()
			break
		case LEVEL_ALERT:
			Lock.Lock()
			An.CodeAlert++
			Lock.Unlock()
			break
		case LEVEL_CRITICAL:
			Lock.Lock()
			An.CodeCritical++
			Lock.Unlock()
			break
		}
	}
}

func (Log *LogService) getLogPath(level string) string {
	format := make([]string, 0)
	if strings.Contains(Cf.Log.FormatType, "time") {
		format = append(format, helper.TimeFormat(Cf.Log.Format, 0))
	}

	if strings.Contains(Cf.Log.FormatType, "level") {
		format = append(format, level)
	}

	return helper.GetPathJoin(Cf.Log.Path, strings.Join(format, ".")+".log")
}

func (Log *LogService)setLogDir() {
	if !strings.HasPrefix(Cf.Log.Path, "/") {
		Cf.Log.Path = helper.GetPathJoin(Cf.AppPath, Cf.Log.Path)
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

func (Log *LogService) WriteAppend(fileName string, content string) {
	LockPosition.Lock()
	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
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