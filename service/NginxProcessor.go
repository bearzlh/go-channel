package service

import (
	"fmt"
	"github.com/gofrs/uuid"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"
	"workerChannel/helper"
	"workerChannel/object"
)

var nginxLineNumber int64
var nginxLineLock sync.Mutex

var nginxPostLineNumber int64
var nginxPostLineLock sync.Mutex

const NginxRegex = `^\[(\d+)\] (\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}) - - \[(\d{1,2})/(\w+)/(\d{4}):(\d{2}):(\d{2}):(\d{2}) \+0800\] "(.*?)" "(\w+) (.*?) HTTP/1.1 status:(\d+) cost:(.*?) php:(.*?) (\d+)" "(.*?)" "(.*?)" "(.*?)"(.*?)`

func NginxProcessLine(Rp ReadPath) {
	tail := Tail[Rp.Type]
	for line := range tail.Lines {
		nginxLine := IncreaseNginxLineNumber()
		//添加当前读取的位置
		text := fmt.Sprintf("[%d] "+line.Text, nginxLine)
		Lock.Lock()
		An.LineCount++
		Lock.Unlock()
		if len(helper.RegexpMatch(text, NginxRegex)) > 0 {
			//记录当前id
			NginxLineToJob(text, Rp.Type)
		} else {
			L.Debug("nginx log format not matched"+line.Text, LEVEL_DEBUG)
		}
	}
}

func NginxLineToJob(text string, Type string) {
	jobId, _ := uuid.NewV4()
	Lock.Lock()
	An.JobProcessing++
	An.JobCount++
	Lock.Unlock()
	SetMap(jobId.String(), LineItem{Type, []string{text}})
	JobQueue <- jobId.String()
}

func SetNginxPostLineNumber(line int64, cover bool) int64 {
	nginxPostLineLock.Lock()
	defer nginxPostLineLock.Unlock()
	if cover {
		nginxPostLineNumber = line
	} else {
		if line > nginxPostLineNumber {
			nginxPostLineNumber = line
		}
	}
	return nginxPostLineNumber
}

func GetNginxPostLineNumber() int64 {
	nginxPostLineLock.Lock()
	defer nginxPostLineLock.Unlock()
	return nginxPostLineNumber
}


func SetNginxLineNumber(line int64) int64 {
	nginxLineLock.Lock()
	defer nginxLineLock.Unlock()
	nginxLineNumber = line
	return nginxLineNumber
}

func IncreaseNginxLineNumber() int64 {
	nginxLineLock.Lock()
	defer nginxLineLock.Unlock()
	nginxLineNumber++
	return nginxLineNumber
}

func GetNginxMsg(lines []string, pm *object.NginxMsg)  {
	for _, data := range lines {
		ProcessNginxMsg(pm, data)
	}
}

func ProcessNginxMsg(pm *object.NginxMsg, data string) {
	msgMatch := helper.RegexpMatch(data, NginxRegex)
	dateMap := map[string]string{"Jan": "01", "Feb": "02", "Mar": "03", "Apr": "04", "May": "05", "Jun": "06", "Jul": "07", "Aug": "08", "Sep": "09", "Oct": "10", "Nov": "11", "Dec": "12"}
	if len(msgMatch) > 0 {
		pm.LogType = "access"
		pm.LogLine, _ = strconv.ParseInt(string(msgMatch[1]), 10, 64)
		pm.Remote = string(msgMatch[2])
		timeFormat := string(msgMatch[3]) + "-" + dateMap[string(msgMatch[4])] + "-" + string(msgMatch[5]) + " " + string(msgMatch[6]) + ":" + string(msgMatch[7]) + ":" + string(msgMatch[8])
		pm.Date = helper.FormatTimeStamp(timeFormat, "02-01-2006 15:04:05")
		pm.Timestamp = fmt.Sprintf("%d", time.Now().Unix())
		pm.ServerPort = string(msgMatch[9])
		pm.Method = string(msgMatch[10])
		pm.Uri = string(msgMatch[11])
		pm.HttpCode = string(msgMatch[12])
		pm.Cost = helper.Round(string(msgMatch[13]), 2)
		pm.UpCost = helper.Round(string(msgMatch[14]), 2)
		pm.SentBytes, _ = strconv.ParseInt(string(msgMatch[15]), 10, 32)
		pm.Referral = string(msgMatch[16])
		pm.Browser = string(msgMatch[17])
		pm.XForward = string(msgMatch[18])
		pm.UpStream = string(msgMatch[19])
		pm.Url = pm.ServerPort + pm.Uri
		u, err := url.Parse(pm.Url)
		if err == nil {
			if len(u.Query()) > 0 {
				NginxQueryProcess(u.Query(), pm)
			}
			pm.Uri = u.Path
		} else {
			L.Debug("URL解析错误"+err.Error(), LEVEL_ERROR)
		}
		//pm.DomainPort = u.Host
		pm.HostName, _ = os.Hostname()
		pm.AppId = GetAppIdFromHostName(pm.HostName)
		pm.Tag = "nginx." + pm.AppId + ".info"
		pm.LogType = "info"
	}
}

//参数查询
func NginxQueryProcess(values url.Values, msg *object.NginxMsg) {
	for field, list := range values {
		msg.Query = append(msg.Query, object.Query{field, list[0]})
	}
}