package service

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"workerChannel/helper"
	"workerChannel/object"
)

var phpLineNumber int64
var phpLineLock sync.Mutex

var phpPostLineNumber int64
var phpPostLineLock sync.Mutex

const PhpFirstLineRegex = `^\[(\d+)\] ([[:alnum:]]{13}) \[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})] (\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}) (GET|POST) (.*)`
const PhpMsgRegex = `^\[(\d+)\] ([[:alnum:]]{13}) \[ (\w+) ] (.*)`
const PhpAnRegex = `^\[(\d+)\] [[:alnum:]]{13} \[ \w+ \]  \[运行时间：(\d+\.\d+)s\]\[吞吐率：.*?\] \[内存消耗：(.*?)kb\] \[文件加载：(\d+)\]`

func PhpProcessLine(Rp ReadPath) {
	var currentId string
	tail := Tail[Rp.Type]
	for line := range tail.Lines {
		phpLine := IncreasePhpLineNumber()
		Lock.Lock()
		An.TimeEnd = time.Now().Unix()
		An.TimeWork = helper.FormatTime(An.TimeEnd - An.TimeStart)
		An.LineCount++
		Lock.Unlock()
		text := fmt.Sprintf("[%d] " + line.Text, phpLine)
		//记录当前id
		currentId = PhpLineToJob(text, Rp.Type, currentId)
	}
}

func PhpLineToJob(text string, Type string, currentId string) string {
	check := helper.RegexpMatch(text, `^\[\d+\] ([[:alnum:]]{13}) `)
	id := ""

	//如果不满足行规则则置空id，id为空则不进行任务处理
	if len(check) != 0 {
		id = string(check[1])
	}

	//如果不满足行规则则匹配到上一个行任务
	if currentId != "" && len(check) == 0 {
		id = currentId
	}

	//任务处理，并计时，规定时间后处理任务
	if item, ok := GetMap(id); !ok && id != "" && len(check) != 0 {
		if len(helper.RegexpMatch(text, PhpFirstLineRegex)) > 0 {
			item = LineItem{Type, []string{text}}
			SetMap(id, item)
			go func(jobId string) {
				t := time.NewTimer(time.Second * time.Duration(Cf.Time))
				select {
				case <-t.C:
					Lock.Lock()
					An.JobCount++
					An.JobProcessing++
					Lock.Unlock()
					JobQueue <- jobId
				}
			}(id)
		} else {
			//重启时移动到上次请求的开始位置，不包含首行
			//L.Debug("not find first line->"+id, LEVEL_DEBUG)
		}
	}else{
		item, ok := GetMap(id)
		if ok {
			item.List = append(item.List, text)
			SetMap(id, item)
		} else {
			//首行为非请求第一行
			//L.Debug("not find request head->"+id, LEVEL_DEBUG)
		}
	}

	//记录当前id
	return id
}

func CheckValid(msg *object.PhpMsg) bool {
	return msg.Xid != ""
}

func GetPhpMsg(lines []string, pm *object.PhpMsg, collect bool) {
	for _, data := range lines {
		MsgAddContent(pm, data, collect)
	}
}

//初始化消息主体
func MsgAddContent(p *object.PhpMsg, line string, collect bool) {
	s := helper.RegexpMatch(line, PhpFirstLineRegex)
	if collect {
		s1 := helper.RegexpMatch(line, PhpAnRegex)
		if len(s1) > 0 {
			p.WorkFile, _ = strconv.Atoi(string(s1[4]))
			memory, err := strconv.ParseFloat(strings.Replace(string(s1[3]), ",", "", 2), 32)
			if err == nil {
				p.WorkMemory = int(memory)
			} else {
				p.WorkMemory = 0
			}
			timeS, err := strconv.ParseFloat(string(s1[2]), 32)
			if err == nil {
				timeMs := timeS * 1000
				p.WorkTime = int(timeMs)
			} else {
				p.WorkTime = 0
			}
		}
	}
	if len(s) > 0 {
		p.LogLine, _ = strconv.ParseInt(string(s[1]), 10, 64)
		p.Xid = string(s[2])
		p.Date = helper.FormatTimeStamp(string(s[3]), "")
		p.Remote = string(s[4])
		p.Method = string(s[5])
		p.Url = strings.TrimSpace(string(s[6]))
		p.Timestamp = fmt.Sprintf("%d", time.Now().Unix())
		u, _ := url.Parse(p.Url)
		if len(u.Query()) > 0 {
			QueryProcess(u.Query(), p)
		}
		p.Uri = u.Path
		p.DomainPort = u.Host
		WechatMatch := helper.RegexpMatch(p.DomainPort, `^(wx\w+)\.`)
		if len(WechatMatch) > 0 {
			p.WechatAppId = string(WechatMatch[1])
		}

		p.HostName, _ = os.Hostname()
		p.AppId = GetAppIdFromHostName(p.HostName)
		p.Tag = "php." + p.AppId + ".info"
		p.LogType = "info"
	} else {
		MatchMessage := helper.RegexpMatch(line, PhpMsgRegex)
		if len(MatchMessage) > 0 {
			Message := object.Content{}
			Message.LogLine = string(MatchMessage[1])
			Message.Xid = string(MatchMessage[2])
			Message.LogType = string(MatchMessage[3])
			if string(MatchMessage[3]) == "error" {
				p.Tag = "php." + p.AppId + ".error"
				p.LogType = "error"
			}
			Message.Content = strings.TrimSpace(string(MatchMessage[4]))
			p.Message = append(p.Message, Message)
		} else {
			if len(p.Message) > 0 {
				p.Message[len(p.Message)-1].Content += strings.TrimSpace(line)
			}
		}
	}
}

//参数查询
func QueryProcess(values url.Values, msg *object.PhpMsg) {
	for field, list := range values {
		msg.Query = append(msg.Query, object.Query{field, list[0]})
		switch field {
		case "referral_id":
			msg.ReferralId = list[0]
			break;
		case "book_id":
			msg.BookId = list[0]
			break;
		case "chapter_id":
			msg.ChapterId = list[0]
			break;
		case "agent_id":
			msg.AgentId = list[0]
			break;
		}
	}

	if msg.BookId != "" && msg.ChapterId != "" {
		msg.BookChapterId = msg.BookId + "_" + msg.ChapterId
	}
}

//获取主机id
func GetAppIdFromHostName(HostName string) string {
	msg := []byte(HostName)
	reg := regexp.MustCompile(`[[:alpha:]]+`)
	s := reg.Find(msg)
	return string(s)
}

func GetPositionFile(logType string) string {
	return helper.GetPathJoin(Cf.AppPath, Cf.PositionFile + "_" + logType)
}

func SetPhpLineNumber(line int64) int64 {
	phpLineLock.Lock()
	defer phpLineLock.Unlock()
	phpLineNumber = line
	return phpLineNumber
}

func SetPhpPostLineNumber(line int64, cover bool) int64 {
	phpPostLineLock.Lock()
	defer phpPostLineLock.Unlock()
	if cover {
		phpPostLineNumber = line
	} else {
		if line > phpPostLineNumber {
			phpPostLineNumber = line
		}
	}

	return phpPostLineNumber
}

func GetPhpPostLineNumber() int64 {
	phpPostLineLock.Lock()
	defer phpPostLineLock.Unlock()
	return phpPostLineNumber
}

func IncreasePhpLineNumber() int64 {
	phpLineLock.Lock()
	defer phpLineLock.Unlock()
	phpLineNumber++
	return phpLineNumber
}