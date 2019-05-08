package service

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
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

var phpPostLineNumber = int64(0)
var phpPostLineLock sync.Mutex
var LineTime float64
var LineCount float64
var UserTable = "openid recharge user"

const PhpFirstLineRegex = `^\[(\d+)\] ([[:alnum:]]{13}) \[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\] (\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}) (GET|POST|HEAD) (.*)`
const PhpMsgRegex = `^\[(\d+)\] ([[:alnum:]]{13}) \[ (\w+) \] (.*)`
const PhpFrontCookie = `.*? NetType:(\w+) IP:.*? \[(.*?)\|0\|(.*?)\|(.*?)\|(.*?)\] user_id:(\w+)* openid:(.*?)* channel_id:(\w+)* agent_id:(\w+)* referral_id:(\w+)*`
const PhpAdminCookie = `.*?\[(.*?)\|0\|(.*?)\|(.*?)\|(.*?)\] admin_id:(\w+)* group:(\w+)* `
const PhpOrder = `(\w+)_create_order_(\w+)!wxpay_id:(.*?),wxpay_name:.*?,mch_id:.*?,channel_id:(.*?),user_id:(.*?),money:(.*?),good_id:(.*?),out_trade_no:.*?`
const PhpOrderCallback = `(\w+)_callback_(\w+)!wxpay_id:(.*?),channel_id:(.*?),money:(.*?),good_id:(.*?),.*?`

var readPath ReadPath

func PhpProcessLine(Rp ReadPath) {
	L.Debug("日志收集开始", LEVEL_INFO)
	readPath = Rp
	var currentId string
	tail := Tail[Rp.Type]
	LineCount = 0
	LineTime = 0
	for line := range tail.Lines {
		Lock.Lock()
		sleepTime := An.SleepTime
		Lock.Unlock()
		time.Sleep(time.Duration(sleepTime))
		now := time.Now()
		phpLine := IncreasePhpLineNumber()
		Lock.Lock()
		An.LineCount++
		Lock.Unlock()
		text := fmt.Sprintf("[%d] ", phpLine) + line.Text
		//记录当前id
		currentId = PhpLineToJob(text, Rp.Type, currentId)
		LineCount++
		LineTime += float64(time.Now().Sub(now))
		if !Cf.Factory.On {
			L.Debug("日志收集暂停", LEVEL_NOTICE)
			break
		}
	}
}

//依据cpu使用率设置读取日志的休眠时间
func GetSleepTime() {
	go func() {
		for {
			select {
			case <-time.After(time.Second):
				Lock.Lock()
				if An.CpuRate > 0 {
					if An.CpuRate > Cf.Monitor.Cpu {
						An.SleepTime += float64(Cf.Monitor.SleepIntervalNs)
					} else {
						An.SleepTime -= float64(Cf.Monitor.SleepIntervalNs)
					}
				}
				if An.SleepTime < 0 {
					An.SleepTime = 0
				}
				Lock.Unlock()
			}
		}
	}()
}

//为日志行分组
func PhpLineToJob(text string, Type string, preId string) string {
	check := helper.RegexpMatch(text, `^\[\d+\] ([[:alnum:]]{13}) `)
	id := ""

	//如果不满足行规则则置空id，id为空则不进行任务处理
	if len(check) != 0 {
		id = string(check[1])
	}

	//任务处理，并计时，规定时间后处理任务
	if item, ok := GetMap(id); !ok && id != "" {
		item = LineItem{Type, []string{text}}
		SetMap(id, item)
		go func(jobId string) {
			select {
			case <-time.After(time.Second * time.Duration(Cf.PhpTimeWindow)):
				Lock.Lock()
				An.JobCount++
				An.JobProcessing++
				Lock.Unlock()
				JobQueue <- jobId
			}
		}(id)
	} else {
		item, ok := GetMap(id)
		if ok {
			item.List = append(item.List, text)
			SetMap(id, item)
		} else {
			if item, ok := GetMap(preId); ok {
				item.List = append(item.List, text)
				SetMap(preId, item)
			}
		}
	}

	//记录当前id
	if id == "" {
		return preId
	} else {
		return id
	}
}

//检测消息对象是否有效
func CheckValid(msg *object.PhpMsg) bool {
	return msg.Xid != ""
}

//获取php消息对象
func GetPhpMsg(lines []string, pm *object.PhpMsg) {
	for index, data := range lines {
		if index == 0 {
			MsgAddContent(pm, data, true)
		} else {
			MsgAddContent(pm, data, false)
		}
	}
}

//初始化消息主体
func MsgAddContent(p *object.PhpMsg, line string, firstLine bool) {
	if firstLine {
		s := helper.RegexpMatch(line, PhpFirstLineRegex)
		if len(s) > 0 {
			p.Date = helper.FormatTimeStamp(string(s[3]), "")
			if Cf.Recover.From != "" {
				fromTime := helper.FormatTimeStamp(Cf.Recover.From, "")
				toTime := helper.FormatTimeStamp(Cf.Recover.To, "")
				if p.Date < fromTime {
					return
				}
				if p.Date > toTime {
					L.Debug("暂存过期，程序退出", LEVEL_NOTICE)
					cmd := exec.Command("/bin/bash", "-c", "cp "+helper.GetPathJoin(Cf.AppPath, "storage/data") +" /usr/local/postlog/storage/")
					_, err := cmd.Output()
					if err != nil {
						L.Debug(err.Error(), LEVEL_ERROR)
					}
					StopSignal <- os.Interrupt
					return
				}
			}
			p.LogLine, _ = strconv.ParseInt(string(s[1]), 10, 64)
			p.Xid = string(s[2])
			p.Remote = string(s[4])
			p.Method = string(s[5])
			p.Url = strings.TrimSpace(string(s[6]))
			p.Timestamp = fmt.Sprintf("%d", time.Now().Unix())
			u, err := ParseUrl(p.Url)
			if err != nil {
				L.Debug("url parse error"+err.Error()+",url:"+p.Url, LEVEL_ERROR)
			}else{
				//添加get参数
				if strings.Contains(readPath.Pick, "get") {
					if len(u.Query()) > 0 {
						QueryProcess(u.Query(), p)
					}
				}
				p.Uri = u.Path
				p.DomainPort = u.Host
			}
			WechatMatchOld := helper.RegexpMatch(p.DomainPort, `^(wx\w+)\.`)
			if len(WechatMatchOld) > 0 {
				p.WechatAppId = string(WechatMatchOld[1])
			}
			WechatMatchDomain := helper.RegexpMatch(p.DomainPort, `^px-\w+-(\w+)-(\d+)-\w+\..*`)
			if len(WechatMatchDomain) > 0 {
				p.WechatAppId = string(WechatMatchDomain[1])
				p.UserId = string(WechatMatchDomain[2])
			}
			WechatMatchUri := helper.RegexpMatch(p.Uri, `^/api/wechat/mpapi/appid/(\w+)`)
			if len(WechatMatchUri) > 0 {
				p.WechatAppId = string(WechatMatchUri[1])
			}

			p.HostName, _ = os.Hostname()
			p.AppId = GetAppIdFromHostName(p.HostName)
			p.Tag = "php." + p.AppId + ".info"
			p.LogType = "info"
		}
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

			//添加cookie参数
			if strings.Contains(readPath.Pick, "cookie") {
				if len(Message.Content) >= 3 && Message.Content[0:3] == "OS:" {
					if (len(p.Uri) >= 6 && p.Uri[0:6] == "/index") || p.Uri == "/" {
						res := helper.RegexpMatch(Message.Content, PhpFrontCookie)
						if len(res) > 0 {
							p.Access = string(res[1])
							p.Country = string(res[2])
							p.Province = string(res[3])
							p.City = string(res[4])
							p.Operator = string(res[5])
							if string(res[6]) != "" {
								p.UserId = string(res[6])
							}
							if string(res[7]) != "" {
								p.OpenId = string(res[7])
							}
							p.ChannelId = string(res[8])
							p.AgentId = string(res[9])
						}
					}
					if len(p.Uri) >= 6 && p.Uri[0:6] == "/admin" {
						res := helper.RegexpMatch(Message.Content, PhpAdminCookie)
						if len(res) > 0 {
							p.Country = string(res[1])
							p.Province = string(res[2])
							p.City = string(res[3])
							p.Operator = string(res[4])
							p.AdminId = string(res[5])
							p.Group = string(res[6])
						}
					}
				}
			}

			//添加订单参数
			if strings.Contains(readPath.Pick, "order") {
				var res [][]byte
				if strings.Contains(p.Uri, "/api/recharge/pay") {
					res = helper.RegexpMatch(Message.Content, PhpOrder)
					if len(res) > 0 {
						p.PayId = string(res[3])
						p.PayType = string(res[1])
						p.PayStatus = "create_" + string(res[2])
						p.ChannelId = string(res[4])
						p.UserId = string(res[5])
						p.Money = helper.Round(string(res[6]), 2)
						p.GoodId = string(res[7])
					}
				} else if strings.Contains(p.DomainPort, "callback") {
					res = helper.RegexpMatch(Message.Content, PhpOrderCallback)
					if len(res) > 0 {
						p.PayId = string(res[3])
						p.PayType = string(res[1])
						p.PayStatus = "callback_" + string(res[2])
						p.ChannelId = string(res[4])
						p.Money = helper.Round(string(res[5]), 2)
						p.GoodId = string(res[6])
					}
				}
			}

			//采集用户信息
			if strings.Contains(readPath.Pick, "user") {
				keywords := "get_db_connect"
				if len(Message.Content) >= len(keywords) && Message.Content[0:len(keywords)] == keywords {
					res := helper.RegexpMatch(Message.Content, `get_db_connect table:(\w+) params:(\d+)`)
					if len(res) > 0 {
						tableName := string(res[1])
						if strings.Contains(UserTable, tableName) {
							p.UserId = string(res[2])
						}
					}
				}
			}
		} else {
			if len(p.Message) > 0 {
				Match := helper.RegexpMatch(line, `^\[\d+\] (.*)`)
				p.Message[len(p.Message)-1].Content += strings.TrimSpace(string(Match[1]))
			}
		}
	}
}

//获取GET参数
func QueryProcess(values url.Values, msg *object.PhpMsg) {
	for field, list := range values {
		msg.Query = append(msg.Query, object.Query{Key: field, Value: list[0]})
		switch field {
		case "referral_id":
			msg.ReferralId = list[0]
			break
		case "book_id":
			msg.BookId = list[0]
			break
		case "chapter_id":
			msg.ChapterId = list[0]
			break
		case "agent_id":
			msg.AgentId = list[0]
			break
		case "user_id":
			msg.UserId = list[0]
			break
		case "openid":
			msg.OpenId = list[0]
			break
		}
	}

	if msg.BookId != "" && msg.ChapterId != "" {
		msg.BookChapterId = msg.BookId + "_" + msg.ChapterId
	}
}

//获取主机id
func GetAppIdFromHostName(HostName string) string {
	if Cf.AppId != "" {
		return Cf.AppId
	} else {
		msg := []byte(HostName)
		reg := regexp.MustCompile(`[[:alpha:]]+`)
		s := reg.Find(msg)
		return string(s)
	}
}

//获取日志读取位置文件
func GetPositionFile(logType string) string {
	return helper.GetPathJoin(Cf.AppPath, ".position_" + logType)
}

//设置php读取行
func SetPhpLineNumber(line int64) int64 {
	phpLineLock.Lock()
	defer phpLineLock.Unlock()
	phpLineNumber = line
	return phpLineNumber
}

//设置php发送行
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

//获取php发送行
func GetPhpPostLineNumber() int64 {
	phpPostLineLock.Lock()
	defer phpPostLineLock.Unlock()
	return phpPostLineNumber
}

//提高php读取行
func IncreasePhpLineNumber() int64 {
	phpLineLock.Lock()
	defer phpLineLock.Unlock()
	phpLineNumber++
	return phpLineNumber
}

//解析url
func ParseUrl(urlStr string) (*url.URL, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		L.Debug("url parse error"+err.Error()+",url:"+urlStr, LEVEL_NOTICE)
		if strings.Contains(urlStr, "%!") {
			L.Debug("url parse save"+",url:"+urlStr, LEVEL_INFO)
			urlSplit := strings.Split(urlStr, "%!")
			return ParseUrl(urlSplit[0])
		}
	}

	return u, err
}