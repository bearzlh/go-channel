package service

import (
	"fmt"
	"io"
	"os"
	"path"
	"runtime/debug"
	"strings"
	"time"
	"workerChannel/helper"
	"workerChannel/object"
)

//日志类型
const (
	LEVEL_DEBUG    = "debug"    //细节信息
	LEVEL_INFO     = "info"     //运行时信息
	LEVEL_NOTICE   = "notice"   //发行变化的关键信息
	LEVEL_ERROR    = "error"    //影响正常功能，只影响当前运行线程
	LEVEL_ALERT    = "alert"    //严重影响正常功能，影响整个进程的运行
	LEVEL_CRITICAL = "critical" //系统问题，进程退出
)

//日志文件指针
var FilePoint *os.File
//日志名称
var FileName string

//日志等级
var LogLevel = map[string]int8{
	LEVEL_DEBUG:    1,
	LEVEL_INFO:     2,
	LEVEL_NOTICE:   3,
	LEVEL_ERROR:    4,
	LEVEL_ALERT:    5,
	LEVEL_CRITICAL: 6,
}

//日志配置接口
type LogConfigInterface interface {
	GetLevel() string
	GetFormatTime() string
	GetFormatLevel() bool
	GetPath() string
}

type LogService struct {
	Config LogConfigInterface
}

//全局
var L *LogService

func GetLog(config LogConfigInterface) {
	L = new(LogService)
	L.Config = config
}

//打日志
func (Log *LogService) Debug(msg string, level string) {
	logLevel := int8(0)
	if _, ok := LogLevel[Log.Config.GetLevel()]; ok {
		logLevel = LogLevel[Log.Config.GetLevel()]
	}

	currentLevel := LogLevel[level]

	if currentLevel >= logLevel {
		logFile := Log.getLogPath(level)
		err := helper.Mkdir(path.Dir(logFile))
		if err != nil {
			Log.outPut(fmt.Sprintf("%s\n", err))
			ExitProgramme(os.Interrupt)
		}
		content := Log.getMsg(msg, level)
		Log.WriteLog(logFile, content)
		if currentLevel >= 4 {
			errorContent := string(debug.Stack())
			Log.WriteLog(logFile, errorContent)
		}
		switch level {
		case LEVEL_ERROR:
			object.CodeError++
			break
		case LEVEL_ALERT:
			object.CodeAlert++
			break
		case LEVEL_CRITICAL:
			object.CodeCritical++
			break
		}
	}
}

func (Log *LogService) getLogPath(level string) string {
	format := make([]string, 0)
	if Log.Config.GetFormatTime() != "" {
		format = append(format, helper.TimeFormat(Log.Config.GetFormatTime(), 0))
	}

	if Log.Config.GetFormatLevel() {
		format = append(format, level)
	}

	return helper.GetPathJoin(Log.Config.GetPath(), strings.Join(format, ".")+".log")
}

//日志信息格式化
func (Log *LogService) getMsg(msg string, level string) string {
	Nano := time.Now().Nanosecond() / 1000000
	msg = fmt.Sprintf("level:[%s]\ttime:[%s.%d]\tpid:[%d]\tmsg:[%s]", level, time.Now().Format("2006-01-02 15:04:05"), Nano, os.Getpid(), msg)
	return msg + "\n"
}

func (Log *LogService) outPut(msg string) {
	fmt.Print(msg)
}

//写日志
func (Log *LogService) WriteLog(fileName string, content string) {
	Log.outPut(content)
	if FileName == "" || fileName != FileName {
		FileName = fileName
		FilePoint, _ = os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	}
	_, err := io.WriteString(FilePoint, content)
	if err != nil {
		Log.outPut(fmt.Sprintf("%s", err.Error()))
	}
}
