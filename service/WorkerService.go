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
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	"workerChannel/helper"
	"workerChannel/object"
)

type Analysis struct {
	LineCount int64 `json:"line_count"`

	//请求统计
	JobCount      int64 `json:"job_count"`
	JobQueue      int   `json:"job_queue"`
	JobProcessing int64 `json:"job_processing"`
	JobSuccess    int64 `json:"job_success"`
	JobErrorCount int64 `json:"job_error"`

	//运行时间
	TimeStart    int64  `json:"time_start"`
	TimeStartStr string `json:"time_start_str"`
	TimeEnd      int64  `json:"time_end"`
	TimeEndStr   string `json:"time_end_str"`
	TimeWork     string `json:"time_work"`

	HeapMemoryUsed uint64 `json:"memory_used_M"`
	SysMemoryUsed  uint64 `json:"sys_memory_used_M"`

	BatchLength int `json:"batch_length"`

	//workJobs
	WorkerMap []*Worker `json:"worker_map"`
}

type Worker struct {
	ID        string    `json:"id"`
	IsWorking bool      `json:"is_working"`
	IsQuit    bool `json:"is_quit"`
}

type workerPool struct {
	WorkerList []*Worker
}

type LineItem struct{
	Type string `json:"type"`
	List []string `json:"list"`
}

var LineMap map[string]LineItem

var An Analysis
var WorkPool *workerPool
var JobQueue = make(chan string, 10000)
var Tail map[string]*tail.Tail
var Lock *sync.Mutex
var MapLock *sync.Mutex
var StopSignal = make(chan os.Signal)

//业务处理
func (w *Worker) handleJob(jobId string) {
	L.Debug(fmt.Sprintf("Job doing,id=>%s", jobId), LEVEL_DEBUG)
	var JobError int64 = 0
	if item, ok := GetMap(jobId);ok {
		if item.Type == "php" {
			Msg := object.PhpMsg{}
			GetPhpMsg(item.List, &Msg)
			if CheckValid(&Msg) {
				//批量发送
				if Cf.Msg.IsBatch {
					Es.BuckAdd(Msg)
				} else {
					Es.PostAdd(Msg)
				}
				L.Debug("content=>"+MsgToJson(Msg), LEVEL_DEBUG)
			} else {
				JobError = 1
			}
		} else {
			Msg := object.NginxMsg{}
			GetNginxMsg(item.List, &Msg)
			//批量发送
			if Cf.Msg.IsBatch {
				Es.BuckAdd(Msg)
			} else {
				Es.PostAdd(Msg)
			}
			L.Debug("content=>"+MsgToJson(Msg), LEVEL_DEBUG)
		}

		Lock.Lock()
		An.JobErrorCount += JobError
		An.JobProcessing--
		Lock.Unlock()
		DelMap(jobId)
	}else{
		L.Debug("job error,for id=>"+jobId, LEVEL_ERROR)
	}
}

//初始化任务队列
func (w *Worker) Start() {
	go func() {
		L.Debug(fmt.Sprintf("worker %s waiting", w.ID), LEVEL_DEBUG)
		for {
			select {
			case jobID := <-JobQueue:
				L.Debug(fmt.Sprintf("worker: %s, will handle job: %s", w.ID, jobID), LEVEL_DEBUG)
				w.IsWorking = true
				w.handleJob(jobID)
				w.IsWorking = false
				if w.IsQuit {
					L.Debug(fmt.Sprintf("worker: %s, will quit", w.ID), LEVEL_DEBUG)
					break
				}
			}
		}
	}()
}

func NewWorker() {
	id, _ := uuid.NewV4()
	worker := &Worker{ID: id.String(), IsWorking: false}
	worker.Start()
	WorkPool.WorkerList = append(WorkPool.WorkerList, worker)
	L.Debug(fmt.Sprintf("worker %s started", worker.ID), LEVEL_DEBUG)
}

//初始化工厂
func InitWorkPool() {
	if An.TimeStart == 0 {
		An.TimeStart = time.Now().Unix()
	}
	Lock = new(sync.Mutex)
	MapLock = new(sync.Mutex)
	LineMap = make(map[string]LineItem)
	Tail = make(map[string]*tail.Tail)
	SetWorker(Cf.Factory.WorkerInit)
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

//执行下一个文件
func TailNextFile(FileName string, Rp ReadPath) {
	L.Debug("check " + Rp.Type, LEVEL_DEBUG)
	f := PhpProcessLine
	switch Rp.Type {
	case "php":
		f = PhpProcessLine
		break
	case "nginx":
		f = NginxProcessLine
		break;
	}
	if Tail[Rp.Type] != nil && Tail[Rp.Type].Filename != "" {
		if Tail[Rp.Type].Filename != FileName {
			L.Debug("file changed:"+Tail[Rp.Type].Filename+"->"+FileName, LEVEL_DEBUG)
			Tail[Rp.Type].Cleanup()
			err := Tail[Rp.Type].Stop()
			if err != nil {
				L.Debug("file stop error:"+err.Error(), LEVEL_DEBUG)
			}
			go func() {
				TailFile(FileName, Rp, f)
			}()
		} else {
			L.Debug("file not changed", LEVEL_DEBUG)
		}
	} else {
		go func() {
			L.Debug("file init->"+FileName, LEVEL_DEBUG)
			TailFile(FileName, Rp, f)
		}()
	}
}

//执行查询
func TailFile(FileName string, Rp ReadPath, f func(ReadPath)) {
	L.Debug("current_file=>"+FileName, LEVEL_DEBUG)
	whence := io.SeekCurrent
	var position, currentLine int64 = 0, 0
	if Rp.Continue {
		//获取并使用之前执行的位置
		positionObj := GetPosition(GetPositionFile(Rp.Type))
		if positionObj.File == FileName {
			currentLine = positionObj.Line
			position = GetPositionFromFileLine(FileName, currentLine)
			currentLine = GetFileLineFromPosition(FileName, position)
		}
	} else {
		currentLine = GetFileEndLine(FileName)
		whence = io.SeekEnd
	}

	if Rp.Type == "php" {
		L.Debug(fmt.Sprintf("get php line %d", currentLine), LEVEL_DEBUG)
		SetPhpLineNumber(currentLine)
		SetPhpPostLineNumber(currentLine, true)
	} else {
		L.Debug(fmt.Sprintf("get nginx line %d", currentLine), LEVEL_DEBUG)
		SetNginxLineNumber(currentLine)
		SetNginxPostLineNumber(currentLine, true)
	}
	seek := tail.SeekInfo{Offset: position, Whence: whence}
	//获取文件流
	tailFile, err := tail.TailFile(FileName, tail.Config{Follow: true, Location: &seek})
	if err != nil {
		L.Debug(err.Error(), LEVEL_ERROR)
		return
	}
	Lock.Lock()
	Tail[Rp.Type] = tailFile
	Lock.Unlock()

	f(Rp)
}

func GetLogFile(logType ReadPath, time int64) string {
	return helper.GetPathJoin(logType.Dir, helper.TimeFormat(logType.TimeFormat, time) + logType.Suffix)
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
func MsgToJson(msg object.MsgInterface) string {
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
func GetAnalysis() []byte {
	if WorkPool != nil {
		An.WorkerMap = WorkPool.WorkerList
	}

	An.JobQueue = len(JobQueue)
	An.TimeStartStr = helper.TimeFormat("Y-m-d H:i:s", An.TimeStart)
	An.TimeEndStr = helper.TimeFormat("Y-m-d H:i:s", An.TimeEnd)
	An.TimeWork = helper.FormatTime(An.TimeEnd - An.TimeStart)
	memStat := new(runtime.MemStats)
	runtime.ReadMemStats(memStat)
	An.HeapMemoryUsed = memStat.Alloc / 1024 / 1024
	An.SysMemoryUsed = memStat.Sys / 1024 / 1024
	An.BatchLength = len(BuckDoc)

	jsonData, err := json.Marshal(An)
	if err != nil {
		L.Debug(err.Error(), LEVEL_ERROR)
	}
	return jsonData
}

//获取统计信息
func SetAnalysis() {
	file, _ := ioutil.ReadFile(helper.GetPathJoin(Cf.AppPath, ".analysis"))
	if len(file) == 0 {
		L.Debug("analysis file empty", LEVEL_DEBUG)
		return
	}
	An = Analysis{}
	err1 := json.Unmarshal(file, &An)
	if err1 != nil {
		L.Debug("analysis unmarshal error"+err1.Error(), LEVEL_ERROR)
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
		if nextTime > time.Now().Unix() {
			break;
		}
	}

	return resFile
}

func ExitProgramme(s os.Signal) {
	StopSignal <- s
}