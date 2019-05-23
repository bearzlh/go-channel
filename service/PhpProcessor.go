package service

import (
	"encoding/json"
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

type Processor struct {
	Rp                ReadPath
	phpLineNumber     int64
	phpPostLineNumber     int64
}

var phpPostLineLock = new(sync.Mutex)

var UserTable = "openid recharge user"

var MsgLock = new(sync.Mutex)

const PhpFirstLineRegex = `^\[(\d+)\] ([[:alnum:]]{13}) \[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\] (\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}) (GET|POST|HEAD|OPTIONS|PUT|DELETE|TRACE|CONNECT) (.*)`
const PhpMsgRegex = `^\[(\d+)\] ([[:alnum:]]{13}) \[ (\w+) \] (.*)`
const PhpFrontCookie = `.*? NetType:(\w+) IP:.*? \[(.*?)\|0\|(.*?)\|(.*?)\|(.*?)\] user_id:(\w+)* openid:(.*?)* channel_id:(\w+)* agent_id:(\w+)* referral_id:(\w+)*`
const PhpAdminCookie = `.*?\[(.*?)\|0\|(.*?)\|(.*?)\|(.*?)\] admin_id:(\w+)* group:(\w+)* `
const PhpOrder = `(\w+)_create_order_(\w+)!wxpay_id:(.*?),wxpay_name:.*?,mch_id:.*?,channel_id:(.*?),user_id:(.*?),money:(.*?),good_id:(.*?),out_trade_no:.*?`
const PhpOrderCallback = `(\w+)_callback_(\w+)!wxpay_id:(.*?),channel_id:(.*?),money:(.*?),good_id:(.*?),.*?`

func (PP *Processor) ProcessLine() {
	L.Debug("日志收集开始"+PP.Rp.Type, LEVEL_INFO)
	tail := Tail[PP.Rp.Type]
	text := ""
	for line := range tail.Lines {
		phpLine := PP.IncreaseLineNumber()
		check := helper.RegexpMatch(line.Text, `^([[:alnum:]]{13}) `)
		cancel := make(chan bool, 1)
		if len(check) > 0 {
			PP.LineToJob(text)
			text = fmt.Sprintf("[%d] "+line.Text, phpLine)
		} else {
			cancel<-true
			text += line.Text
		}
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
				GetCpu()
				cpuRate := object.CpuRate
				if cpuRate > 0 {
					if cpuRate > Cf.Monitor.Cpu {
						object.SleepTime += float64(Cf.Monitor.SleepIntervalNs)
					} else {
						object.SleepTime -= float64(Cf.Monitor.SleepIntervalNs)
					}
				}
				if object.SleepTime < 0 {
					object.SleepTime = 0
				}
			}
		}
	}()
}

//为日志行分组
func (PP *Processor) LineToJob(text string) {
	check := helper.RegexpMatch(text, `^\[\d+\] ([[:alnum:]]{13}) `)
	if len(check) == 0 {
		return
	}
	id := string(check[1])

	//任务处理，并计时，规定时间后处理任务
	if item, ok := GetMap(id); !ok && id != "" {
		item = LineItem{PP.Rp.Type, []string{text}}
		SetMap(id, item)
		go func(jobId string) {
			select {
			case <-time.After(time.Second * time.Duration(Cf.PhpTimeWindow)):
				object.JobCount++
				object.JobProcessing++
				JobQueue <- jobId
			}
		}(id)
	} else {
		item, ok := GetMap(id)
		if ok {
			item.List = append(item.List, text)
			SetMap(id, item)
		} else {
			SetMap(id, item)
		}
	}
}

//检测消息对象是否有效
func CheckValid(msg *object.PhpMsg) bool {
	return msg.Xid != ""
}

//获取php消息对象
func (PP *Processor) GetPhpMsg(lines []string, pm *object.PhpMsg) {
	if len(lines) <= 1 {
		return
	}
	L.Debug(fmt.Sprintf("line len:%d", len(lines)), LEVEL_DEBUG)
	msgCh := make(chan int, len(lines))
	nextCh := make(chan int, 1)
	pm.Message = make([]object.Content, len(lines)-1)
	object.TimeEnd = time.Now().Unix()
	go func() {
		for index, data := range lines {
			if index == 0 {
				PP.getMessageFirstLine(pm, data)
				msgCh<-0
			} else {
				go func(msgCh, nextCh chan int, index int, data string) {
					PP.getMessage(pm, data, index, msgCh, nextCh)
				}(msgCh, nextCh, index, data)
			}
		}
	}()
	<-nextCh
}

func (PP *Processor) getMessageFirstLine(p *object.PhpMsg, line string) {
	s := helper.RegexpMatch(line, PhpFirstLineRegex)
	if len(s) > 0 {
		Date := helper.FormatTimeStamp(string(s[3]), "")
		LogLine, _ := strconv.ParseInt(string(s[1]), 10, 64)
		Xid := string(s[2])
		Remote := string(s[4])
		Country, Province, City, Jingwei := IP.GetLocation(Remote)
		Method := string(s[5])
		Url := strings.TrimSpace(string(s[6]))
		Timestamp := fmt.Sprintf("%d", time.Now().Unix())
		u, err := ParseUrl(Url)
		Uri, DomainPort := "", ""
		if err != nil {
			L.Debug("url parse error"+err.Error()+",url:"+Url, LEVEL_ERROR)
		} else {
			//添加get参数
			if strings.Contains(PP.Rp.Pick, "get") {
				if len(u.Query()) > 0 {
					QueryProcess(u.Query(), p)
				}
			}
			Uri = u.Path
			DomainPort = u.Host
		}
		WechatAppId, UserId, AppId := "", "", ""
		WechatMatchOld := helper.RegexpMatch(DomainPort, `^(wx\w+)\.`)
		if len(WechatMatchOld) > 0 {
			WechatAppId = string(WechatMatchOld[1])
		}
		WechatMatchDomain := helper.RegexpMatch(DomainPort, `^px-\w+-(\w+)-(\d+)-\w+\..*`)
		if len(WechatMatchDomain) > 0 {
			WechatAppId = string(WechatMatchDomain[1])
			UserId = string(WechatMatchDomain[2])
		}
		WechatMatchUri := helper.RegexpMatch(Uri, `^/api/wechat/mpapi/appid/(\w+)`)
		if len(WechatMatchUri) > 0 {
			WechatAppId = string(WechatMatchUri[1])
		}

		HostName, _ := os.Hostname()
		if PP.Rp.AppId != "" {
			AppId = PP.Rp.AppId
		} else {
			AppId = GetAppIdFromHostName(HostName)
		}
		Tag := "php." + AppId + ".info"
		LogType := "info"

		MsgLock.Lock()
		p.Date, p.LogLine, p.Xid, p.Remote, p.Country, p.Province, p.City, p.Jingwei, p.Method, p.Url, p.Timestamp, p.Uri, p.DomainPort, p.WechatAppId, p.UserId, p.AppId, p.HostName, p.Tag, p.LogType = Date, LogLine, Xid, Remote, Country, Province, City, Jingwei, Method, Url, Timestamp, Uri, DomainPort, WechatAppId, UserId, AppId, HostName, Tag, LogType
		MsgLock.Unlock()

		if Cf.Recover.From != "" {
			fromTime := helper.FormatTimeStamp(Cf.Recover.From, "")
			toTime := helper.FormatTimeStamp(Cf.Recover.To, "")
			if Date < fromTime {
				return
			}
			if Date > toTime {
				L.Debug("暂存过期，程序退出", LEVEL_NOTICE)
				cmd := exec.Command("/bin/bash", "-c", "cp "+helper.GetPathJoin(Cf.AppPath, "storage/data")+" /usr/local/postlog/storage/")
				_, err := cmd.Output()
				if err != nil {
					L.Debug(err.Error(), LEVEL_ERROR)
				}
				StopSignal <- os.Interrupt
				return
			}
		}
	}
}

func (PP *Processor) getMessage(p *object.PhpMsg, line string, index int, msgCh, nextCh chan int) {
	MatchMessage := helper.RegexpMatch(line, PhpMsgRegex)
	index -= 1
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
		p.Message[index] = Message

		//添加cookie参数
		if strings.Contains(PP.Rp.Pick, "cookie") {
			if len(Message.Content) >= 3 && Message.Content[0:3] == "OS:" {
				if (len(p.Uri) >= 6 && p.Uri[0:6] == "/index") || p.Uri == "/" {
					res := helper.RegexpMatch(Message.Content, PhpFrontCookie)
					if len(res) > 0 {
						MsgLock.Lock()
						if p.Country == "" {
							p.Country = string(res[2])
						}
						if p.Province == "" {
							p.Province = string(res[3])
						}
						if p.City == "" {
							p.City = string(res[4])
						}
						if string(res[6]) != "" {
							p.UserId = string(res[6])
						}
						if string(res[7]) != "" {
							p.OpenId = string(res[7])
						}
						p.ChannelId, p.AgentId, p.Operator, p.Access = string(res[8]), string(res[9]), string(res[5]), string(res[1])
						MsgLock.Unlock()
					}
				}
				if len(p.Uri) >= 6 && p.Uri[0:6] == "/admin" {
					res := helper.RegexpMatch(Message.Content, PhpAdminCookie)
					if len(res) > 0 {
						MsgLock.Lock()
						p.Country, p.Province, p.City, p.Operator, p.AdminId, p.Group = string(res[1]), string(res[2]), string(res[3]), string(res[4]), string(res[5]), string(res[6])
						MsgLock.Unlock()
					}
				}
			}
		}

		//添加订单参数
		if strings.Contains(PP.Rp.Pick, "order") {
			var res [][]byte
			if strings.Contains(p.Uri, "/api/recharge/pay") {
				res = helper.RegexpMatch(Message.Content, PhpOrder)
				if len(res) > 0 {
					MsgLock.Lock()
					p.PayId, p.PayType, p.PayStatus, p.ChannelId, p.UserId, p.Money, p.GoodId = string(res[3]), string(res[1]), "create_"+string(res[2]), string(res[4]), string(res[5]), helper.Round(string(res[6]), 2), string(res[7])
					MsgLock.Unlock()
				}
			} else if strings.Contains(p.DomainPort, "callback") {
				res = helper.RegexpMatch(Message.Content, PhpOrderCallback)
				if len(res) > 0 {
					MsgLock.Lock()
					p.PayId, p.PayType, p.PayStatus, p.ChannelId, p.Money, p.GoodId = string(res[3]), string(res[1]), "callback_"+string(res[2]), string(res[4]), helper.Round(string(res[5]), 2), string(res[6])
					MsgLock.Unlock()
				}
			}
		}

		//采集用户信息
		if strings.Contains(PP.Rp.Pick, "user") {
			keywords := "get_db_connect"
			if len(Message.Content) >= len(keywords) && Message.Content[0:len(keywords)] == keywords {
				res := helper.RegexpMatch(Message.Content, `get_db_connect table:(\w+) params:(\d+)`)
				if len(res) > 0 {
					tableName := string(res[1])
					if strings.Contains(UserTable, tableName) {
						MsgLock.Lock()
						p.UserId = string(res[2])
						MsgLock.Unlock()
					}
				}
			}
		}

		//采集微信回调信息
		if strings.Contains(PP.Rp.Pick, "wechat") {
			keywords := `[ WeChat ] [ MP ] [ API ] Message: `
			if strings.Contains(p.Uri, "/api/wechat/mpapi/appid/") && strings.Contains(Message.Content, keywords) {
				list := strings.Split(Message.Content, keywords)
				wechatString := strings.Replace(list[1], `\"`, `"`, 100)
				WechatMsg := new(object.WechatMsg)
				err := json.Unmarshal([]byte(wechatString), WechatMsg)
				if err != nil {
					L.Debug("获取微信信息失败"+err.Error(), LEVEL_ERROR)
				} else {
					MsgLock.Lock()
					p.WechatMsg = WechatMsg
					MsgLock.Unlock()
				}
			}
		}
	}else{
		fmt.Printf(line)
	}
	msgCh <- 0
	L.Debug(fmt.Sprintf("xid:%s,msgChlen:%d,len(msgCh):%d", p.Xid, cap(msgCh), len(msgCh)), LEVEL_DEBUG)
	if cap(msgCh) == len(msgCh) {
		nextCh <- 0
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
func (PP *Processor)SetPhpLineNumber(line int64) int64 {
	PP.phpLineNumber = line
	return PP.phpLineNumber
}

//提高php读取行
func (PP *Processor) IncreaseLineNumber() int64 {
	PP.phpLineNumber++
	return PP.phpLineNumber
}

//设置php读取行
func (PP *Processor)SetPhpPostLineNumber(line int64) int64 {
	phpPostLineLock.Lock()
	defer phpPostLineLock.Unlock()
	if line > PP.phpPostLineNumber {
		PP.phpPostLineNumber = line
	}

	return PP.phpPostLineNumber
}

//获取php读取行
func (PP *Processor)GetPhpPostLineNumber() int64 {
	return PP.phpPostLineNumber
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