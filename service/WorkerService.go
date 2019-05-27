package service

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/gofrs/uuid"
	"github.com/hpcloud/tail"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"workerChannel/helper"
	"workerChannel/object"
)

type Analysis struct {
	object.SystemAnalysis

	//workJobs
	WorkerMap []*Worker `json:"worker_map"`

	Cf *ConfigService `json:"cf"`

	BackUpLine int64 `json:"back_up_line"`
}

type Worker struct {
	ID        string `json:"id"`
	IsWorking bool   `json:"is_working"`
	IsQuit    bool   `json:"is_quit"`
}

type workerPool struct {
	WorkerList []*Worker
}

type LineItem struct {
	Type string   `json:"type"`
	List []string `json:"list"`
}

var LineMap map[string]LineItem

var An = Analysis{}
var WorkPool *workerPool
var JobQueue = make(chan string, 10000)
var Tail = make(map[string]*tail.Tail)
var PPList = make(map[string]*Processor)
var MapLock = new(sync.Mutex)
var StopSignal = make(chan os.Signal)
var StopStatus = false

//启动日志监控
func StartWork() {
	if Cf.Factory.On {
		L.Debug("启动日志收集", LEVEL_NOTICE)
		//开始监控日志变化
		for i := 0; i < len(Cf.ReadPath); i++ {

			item := Cf.ReadPath[i]

			if item.On {
				//查看当前文件
				go func() {
					positionObj := GetPosition(GetPositionFile(item.Type))
					L.Debug("当前日志路径"+positionObj.File, LEVEL_NOTICE)
					if positionObj.File != "" {
						TailNextFile(positionObj.File, item)
					} else {
						timeFrom := int64(0)
						if Cf.Recover.From != "" {
							timeFrom = helper.FormatTimeStamp(Cf.Recover.From, "")
						}
						Log := GetLogFile(item, timeFrom)
						TailNextFile(Log, item)
					}
				}()

				if item.TimeFormat != "" {
					object.TimeEnd = time.Now().Unix()
					//定时检查是不是需要切换文件
					go func(i int) {
						t := time.NewTimer(time.Second * 3)
						for {
							//以防配置文件修改后不生效
							item = Cf.ReadPath[i]
							if !Cf.Factory.On || StopStatus {
								L.Debug("日志切换检测停止", LEVEL_NOTICE)
								break
							}
							select {
							case <-t.C:
								L.Debug("日志切换检测", LEVEL_DEBUG)
								t.Reset(time.Second * 3)
								if Tail[item.Type] != nil && Tail[item.Type].Filename != "" {
									nextFile := GetNextFile(item, Tail[item.Type].Filename)
									if nextFile != "" {
										formatEnd := helper.TimeFormat("Y-m-d H:i:s", object.TimeEnd)
										if time.Now().Unix() - object.TimeEnd > int64(Cf.Msg.BatchTimeSecond) && len(Tail[item.Type].Lines) == 0 {
											L.Debug("日志切换生效"+nextFile+"-"+formatEnd, LEVEL_INFO)
											TailNextFile(nextFile, item)
										}
									}
								}
							}
						}
					}(i)
				}
			}
		}
	} else {
		L.Debug("未启动日志收集", LEVEL_NOTICE)
	}
}

//停止日志读取
func StopWork() {
	StopStatus = true
	time.Sleep(time.Second * 2)
	SaveRunTimeStatus()
	for key, value := range Tail {
		StopTailFile(value)
		delete(Tail, key)
	}
	IP.Stop()
}

//检测主机状态并发送统计信息
func CheckHostHealth() {
	go func() {
		TimeFive := time.NewTimer(time.Second * time.Duration(Cf.Monitor.CheckInterval))
		TimeThirty := time.NewTimer(time.Second * time.Duration(Cf.Monitor.PickInterval))
		for {
			select {
			case <-TimeFive.C:
				TimeFive.Reset(time.Second * time.Duration(Cf.Monitor.CheckInterval))
				GetAnalysis(true)
				if object.MemRate > Cf.Monitor.MemRestart {
					L.Debug(fmt.Sprintf("内存使用率超过%f%%，进程重启", Cf.Monitor.MemRestart), LEVEL_ERROR)
					RestartCmd()
				}
			case <-TimeThirty.C:
				TimeThirty.Reset(time.Second * time.Duration(Cf.Monitor.PickInterval))
				msg := new(object.WorkerMsg)
				Ana := GetAnalysis(true)
				err := json.Unmarshal([]byte(Ana), msg)
				if err != nil {
					L.Debug("统计信息解析失败"+err.Error(), LEVEL_ERROR)
				} else {
					object.CodeCritical, object.CodeAlert, object.CodeError = 0, 0, 0
					msg.HostName, _ = os.Hostname()
					msg.AppId = GetAppIdFromHostName(msg.HostName)
					msg.Date = time.Now().Unix()
					L.Debug("发送统计数据", LEVEL_INFO)
					object.JobCount++
					Es.BuckAdd(msg)
				}
			}
		}
	}()
}

//业务处理
func (w *Worker) handleJob(jobId string) {
	L.Debug(fmt.Sprintf("Job doing,id=>%s", jobId), LEVEL_DEBUG)
	if item, ok := GetMap(jobId); ok {
		Msg := object.PhpMsg{}
		PPList[item.Type].GetPhpMsg(item.List, &Msg)
		if CheckValid(&Msg) {
			//批量发送
			Es.BuckAdd(Msg)
			L.Debug("content=>"+MsgToJson(Msg), LEVEL_DEBUG)
		} else {
			L.Debug("xid不存在", LEVEL_NOTICE)
		}

		DelMap(jobId)
		L.Debug(fmt.Sprintf("Job done,id=>%s", jobId), LEVEL_DEBUG)
	} else {
		L.Debug("job error,for id=>"+jobId, LEVEL_INFO)
	}
}

//worker等待工作
func (w *Worker) Start() {
	go func() {
		L.Debug(fmt.Sprintf("worker %s waiting", w.ID), LEVEL_NOTICE)
		for {
			if StopStatus {
				L.Debug("worker stopped:"+w.ID, LEVEL_NOTICE)
				return
			}
			select {
			case jobID := <-JobQueue:
				L.Debug(fmt.Sprintf("worker: %s, will handle job: %s", w.ID, jobID), LEVEL_DEBUG)
				w.IsWorking = true
				w.handleJob(jobID)
				w.IsWorking = false
				object.JobProcessing--
				if w.IsQuit {
					L.Debug(fmt.Sprintf("worker: %s, will quit", w.ID), LEVEL_NOTICE)
					break
				}
			}
		}
	}()
}

//初始化worker
func NewWorker() {
	id, _ := uuid.NewV4()
	worker := &Worker{ID: id.String(), IsWorking: false}
	worker.Start()
	WorkPool.WorkerList = append(WorkPool.WorkerList, worker)
	L.Debug(fmt.Sprintf("worker %s started", worker.ID), LEVEL_NOTICE)
}

//初始化工厂
func InitFactory() {
	if An.TimeStart == 0 {
		An.TimeStart = time.Now().Unix()
	}
	CheckHostHealth()
	LineMap = make(map[string]LineItem)
	SetWorker(Cf.Factory.WorkerInit)
	IP.GetDB()
	object.TimeStart = time.Now().Unix()
	object.TimeEnd = time.Now().Unix()
	object.TimePostEnd = time.Now().Unix()
	object.SleepTime = An.SleepTime
	object.JobProcessing = 0
	object.JobCount = 0
	object.JobSuccess = 0
	object.BackUpLine = An.BackUpLine

	GetSleepTime()
}

//添加工人
func SetWorker(n int) {
	if WorkPool == nil {
		WorkPool = &workerPool{
			WorkerList: make([]*Worker, 0, Cf.Factory.WorkerInit),
		}
	}
	workLen := len(WorkPool.WorkerList)
	if n > workLen {
		for i := 0; i < n-workLen; i++ {
			NewWorker()
		}
	}
	if n < workLen {
		Tmp := WorkPool.WorkerList
		WorkPool = &workerPool{
			WorkerList: make([]*Worker, 0, Cf.Factory.WorkerInit),
		}
		for index, item := range Tmp {
			if index < workLen-n {
				item.IsQuit = true
			} else {
				WorkPool.WorkerList = append(WorkPool.WorkerList, item)
			}
		}
	}

}

//读取下一个文件
func TailNextFile(FileName string, Rp ReadPath) {
	L.Debug("check "+Rp.Type, LEVEL_NOTICE)
	if Tail[Rp.Type] != nil && Tail[Rp.Type].Filename != "" {
		if Tail[Rp.Type].Filename != FileName {
			L.Debug("file changed:"+Tail[Rp.Type].Filename+"->"+FileName, LEVEL_NOTICE)
			StopTailFile(Tail[Rp.Type])
			go func() {
				TailFile(FileName, Rp)
			}()
		} else {
			L.Debug("file not changed", LEVEL_DEBUG)
		}
	} else {
		go func() {
			L.Debug("file init->"+FileName, LEVEL_NOTICE)
			TailFile(FileName, Rp)
		}()
	}
}

//停止日志读取
func StopTailFile(tail *tail.Tail) {
	L.Debug("file will stop tail:"+tail.Filename, LEVEL_NOTICE)
	err := tail.Stop()
	if err != nil {
		L.Debug("file stop error:"+err.Error(), LEVEL_ERROR)
	}
}

//执行查询
func TailFile(FileName string, Rp ReadPath) {
	PPList[Rp.Type] = new(Processor)
	PPList[Rp.Type].Rp = Rp
	PPList[Rp.Type].phpLineNumber = int64(0)
	L.Debug("current_file=>"+FileName, LEVEL_INFO)
	whence := io.SeekCurrent
	var position, currentLine int64 = 0, 0
	if Rp.Continue {
		//获取并使用之前执行的位置
		positionObj := GetPosition(GetPositionFile(Rp.Type))
		if positionObj.File == FileName {
			currentLine = positionObj.Line
			position = GetPositionFromFileLine(FileName, currentLine)
		}
	} else {
		currentLine = GetFileEndLine(FileName)
		whence = io.SeekEnd
	}

	L.Debug(fmt.Sprintf("get php line %d", currentLine), LEVEL_INFO)
	PPList[Rp.Type].SetPhpLineNumber(currentLine)
	seek := tail.SeekInfo{Offset: position, Whence: whence}
	//获取文件流
	tailFile, err := tail.TailFile(FileName, tail.Config{Follow: true, Location: &seek})
	if err != nil {
		L.Debug(err.Error(), LEVEL_ERROR)
		return
	}
	Tail[Rp.Type] = tailFile
	PPList[Rp.Type].ProcessLine()
}

func GetLogFile(logType ReadPath, time int64) string {
	return helper.GetPathJoin(logType.Dir, helper.TimeFormat(logType.TimeFormat, time)+logType.Suffix)
}

func GetMap(id string) (LineItem, bool) {
	MapLock.Lock()
	defer MapLock.Unlock()
	item, ok := LineMap[id]
	return item, ok
}

func SetMap(id string, item LineItem) {
	MapLock.Lock()
	defer MapLock.Unlock()
	LineMap[id] = item
}

func DelMap(id string) {
	MapLock.Lock()
	defer MapLock.Unlock()
	delete(LineMap, id)
	if len(LineMap) == 0 {
		LineMap = nil
		runtime.GC()
		LineMap = make(map[string]LineItem)
	}
}

//转化为json格式
func MsgToJson(msg interface{}) string {
	jsonContent, _ := json.Marshal(msg)
	return string(jsonContent)
}

//获取文件行
func GetFileLineFromPosition(filePath string, fileLimit int64) int64 {
	file, _ := os.Open(filePath)
	rd := bufio.NewReader(file)
	var fileLen, lineNumber int64 = 0, 0
	for {
		line, err := rd.ReadString('\n')
		lineNumber++
		fileLen += int64(len(line))
		if err != nil || io.EOF == err || fileLen >= fileLimit {
			break
		}
	}
	return lineNumber
}

//获取行所在位置
func GetPositionFromFileLine(filePath string, fileLine int64) int64 {
	if fileLine == 0 {
		return 0
	}

	file, _ := os.Open(filePath)
	rd := bufio.NewReader(file)
	var filePos, lineNumber int64 = 0, 0
	for {
		line, err := rd.ReadString('\n')
		lineNumber++
		filePos += int64(len(line))
		if err != nil || io.EOF == err || lineNumber >= fileLine {
			break
		}
	}
	return filePos
}

//获取总文件行数
func GetFileEndLine(filePath string) int64 {
	cmd1 := []string{"wc", filePath}
	cmd2 := []string{"awk", "{print $1}"}
	line := helper.ExecShellPipe(cmd1, cmd2)
	lineNumber, err := strconv.ParseInt(line, 10, 64)
	if err != nil {
		L.Debug("文件行数读取失败,"+err.Error(), LEVEL_ERROR)
		return 0
	}
	return lineNumber
}

//获取统计信息
func GetAnalysis(host bool) []byte {
	if WorkPool != nil {
		An.WorkerMap = WorkPool.WorkerList
	}
	An.TimeStart = object.TimeStart
	An.TimeEnd = object.TimeEnd
	An.TimePostEnd = object.TimePostEnd
	An.SleepTime = helper.RoundFloat(object.SleepTime, 2)
	An.JobProcessing = object.JobProcessing
	An.JobCount = object.JobCount
	An.LineLength = 0
	for _, tailItem := range Tail {
		An.LineLength += len(tailItem.Lines)
	}
	timeSpan := object.JobSuccessTime - An.JobSuccessTime
	if timeSpan > 0 {
		An.JobRate = float64((object.JobSuccess - An.JobSuccess) / timeSpan)
	} else {
		An.JobRate = 0.0
	}
	An.JobSuccess = object.JobSuccess
	An.JobSuccessTime = object.JobSuccessTime
	An.BackUpLine = object.BackUpLine
	An.CodeError = object.CodeError
	An.CodeAlert = object.CodeAlert
	An.CodeCritical = object.CodeCritical

	An.JobQueue = len(JobQueue)
	An.PostCurrent = len(ConcurrentPost)
	An.IpCurrent = len(IPCache)
	An.TimeStartStr = helper.TimeFormat("Y-m-d H:i:s", An.TimeStart)
	An.TimeEndStr = helper.TimeFormat("Y-m-d H:i:s", An.TimeEnd)
	if An.TimeEnd > 0 {
		An.TimeWork = helper.FormatTime(An.TimeEnd - An.TimeStart)
	} else {
		An.TimeWork = "0"
	}
	memStat := new(runtime.MemStats)
	runtime.ReadMemStats(memStat)
	An.HeapMemoryUsed = memStat.Alloc / 1024 / 1024
	An.SysMemoryUsed = memStat.Sys / 1024 / 1024
	An.BatchLength = len(BuckDoc)

	if host {
		GetCpu()
		GetLoad()
		GetMem()
		An.CpuRate = object.CpuRate
		An.Load = object.Load
		An.MemRate = object.MemRate
	}

	An.Cf = Cf
	if An.TimePostEnd > 0 {
		An.TimeDelay = time.Now().Unix() - An.TimePostEnd
	}else{
		An.TimeDelay = 0
	}
	An.TimeDelayStr = helper.FormatTime(An.TimeDelay)
	An.TimePostEndStr = helper.TimeFormat("Y-m-d H:i:s", An.TimePostEnd)

	jsonData, err := json.Marshal(An)
	if err != nil {
		L.Debug(err.Error(), LEVEL_ERROR)
	}
	return jsonData
}

func GetCpu() {
	shellPath := helper.GetPathJoin(Cf.AppPath, "host_info.sh cpu")
	out := exec.Command("/bin/bash", "-c", shellPath)
	content, _ := out.Output()
	value := strings.TrimSpace(string(content))
	object.CpuRate = helper.RoundString(string(value), 2)
}

func GetLoad() {
	shellPath := helper.GetPathJoin(Cf.AppPath, "host_info.sh load")
	out := exec.Command("/bin/bash", "-c", shellPath)
	content, _ := out.Output()
	value := strings.TrimSpace(string(content))
	object.Load = helper.RoundString(string(value), 2)
}

func GetMem() {
	shellPath := helper.GetPathJoin(Cf.AppPath, "host_info.sh memory")
	out := exec.Command("/bin/bash", "-c", shellPath)
	content, _ := out.Output()
	value := strings.TrimSpace(string(content))
	object.MemRate = helper.RoundString(string(value), 2)
}

//获取统计信息
func SetAnalysis() {
	file, _ := ioutil.ReadFile(helper.GetPathJoin(Cf.AppPath, ".analysis"))
	if len(file) == 0 {
		L.Debug("analysis file empty", LEVEL_INFO)
		return
	}
	err := json.Unmarshal(file, &An)
	if err != nil {
		L.Debug("analysis unmarshal error"+err.Error(), LEVEL_ERROR)
	}
}

func GetNextFile(rp ReadPath, currentFile string) string {
	timeLayout := rp.TimeFormat
	layout := helper.FormatToLayout(timeLayout)
	format := strings.Replace(currentFile, rp.Dir, "", -1)
	format = strings.Replace(format, rp.Suffix, "", -1)
	format = strings.Trim(format, "/")
	currentTime := helper.FormatTimeStamp(format, layout)
	nextTime := currentTime
	resFile := ""
	for {
		nextTime += helper.GetMinDuration(timeLayout)
		nextFile := GetLogFile(rp, nextTime)
		if helper.PathExists(nextFile) {
			resFile = nextFile
			break
		}
		endtime := time.Now().Unix()
		if Cf.Recover.To != "" {
			endtime = helper.FormatTimeStamp(Cf.Recover.To, "")
		}
		if nextTime > endtime {
			break
		}
	}

	return resFile
}

func ExitProgramme(s os.Signal) {
	StopSignal <- s
}

//保存状态
func SaveRunTimeStatus() {
	for {
		select {
		case <-time.After(time.Millisecond * 1000):
			L.Debug(fmt.Sprintf("SaveRunTimeStatus,%d,%d,%d", time.Now().Unix(), EsRunning, int64(Cf.Msg.BatchTimeSecond)), LEVEL_NOTICE)
			if time.Now().Unix()-EsRunning > int64(Cf.Msg.BatchTimeSecond) && len(ConcurrentPost) == 0 && len(BuckDoc) == 0 {
				for _, rp := range Cf.ReadPath {
					file := GetPositionFile(rp.Type)
					oTail := Tail[rp.Type]
					PP := PPList[rp.Type]
					L.Debug(file, LEVEL_DEBUG)
					if oTail != nil {
						line := PP.GetPhpPostLineNumber()
						P := object.Position{File: oTail.Filename, Line: line}
						L.Debug(fmt.Sprintf("runtime status save,line +%d", line), LEVEL_INFO)
						SetPosition(file, P)
					}
				}

				fileName := helper.GetPathJoin(Cf.AppPath, ".analysis")
				content := string(GetAnalysis(false))
				err := helper.FilePutContents(fileName, content, false)
				if err != nil {
					L.Debug("保存状态失败"+err.Error(), LEVEL_ERROR)
				}
				return
			}
		}
	}
}
