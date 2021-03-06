package service

import (
	"encoding/json"
	"fmt"
	"github.com/bitly/go-simplejson"
	"net/url"
	"os"
	"os/exec"
	"reflect"
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
	phpPostLineNumber int64
}

var phpPostLineLock = new(sync.Mutex)

var runTimeLock = new(sync.Mutex)

var UserTable = "sign recharge consume user_recently_read recharge user"

var MsgLock = new(sync.Mutex)

const PhpFirstLineRegex = `^\[(\d+)\] (.*?) \[(.*?)\] (.*?) (\w+) (.*)`
const PhpMsgRegex = `^\[(\d+)\] (.*?) \[ (\w+) \] (.*)`
const PhpFrontCookie = `.*? NetType:(\w+) IP:.*? \[(.*?)\|0\|(.*?)\|(.*?)\|(.*?)\] user_id:(\w+)* openid:(.*?)* channel_id:(\w+)* agent_id:(\w+)* referral_id:(\w+)*`
const PhpAdminCookie = `.*?\[(.*?)\|0\|(.*?)\|(.*?)\|(.*?)\] admin_id:(\w+)* group:(\w+)* `
const PhpOrder = `(\w+)_create_order_(\w+)!wxpay_id:(.*?),wxpay_name:.*?,mch_id:.*?,channel_id:(.*?),user_id:(.*?),money:(.*?),good_id:(.*?),out_trade_no:.*?`
const PhpOrderCallback = `(\w+)_callback_(\w+)!wxpay_id:(.*?),channel_id:(.*?),money:(.*?),good_id:(.*?),.*?`

func (PP *Processor) ProcessLine() {
	L.Debug("日志收集开始"+PP.Rp.Type, LEVEL_NOTICE)
	tail := Tail[PP.Rp.Type]
	text := ""
	for line := range tail.Lines {
		time.Sleep(time.Duration(object.SleepTime))
		phpLine := PP.IncreaseLineNumber()
		listStr := make([]string, 0)
		if len(line.Text) > 13 {
			listStr = strings.Split(line.Text[0:14], " ")
		}
		if len(listStr) > 0 && len(listStr[0]) == 13 {
			PP.LineToJob(text)
			text = fmt.Sprintf("[%d] ", phpLine) + line.Text
		} else {
			text += line.Text
		}
		if len(tail.Lines) == 0 {
			PP.LineToJob(text)
			text = ""
		}
		if !Cf.Factory.On {
			L.Debug("日志收集暂停", LEVEL_NOTICE)
			break
		}
	}
	L.Debug("文件结束", LEVEL_NOTICE)
}

//为日志行分组
func (PP *Processor) LineToJob(text string) {
	listStr := strings.Split(text, " ")
	if len(listStr) < 2 || len(listStr[1]) != 13 {
		return
	}
	id := listStr[1]

	//任务处理，并计时，规定时间后处理任务
	if item, ok := GetMap(id); !ok && id != "" {
		item = LineItem{PP.Rp.Type, []string{text}}
		SetMap(id, item)
		go func(jobId string) {
			object.JobCount++
			object.JobProcessing++
			select {
			case <-time.After(time.Second * time.Duration(Cf.PhpTimeWindow)):
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
	msgCh := make(chan int, len(lines))
	nextCh := make(chan int, 1)
	pm.Message = make([]object.Content, len(lines)-1)
	object.TimeEnd = time.Now().Unix()
	go func() {
		for index, data := range lines {
			if index == 0 {
				PP.getMessageFirstLine(pm, data)
				msgCh <- 0
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
	p.RunTime = map[string]float64{}
	p.RequestTag = map[string]string{}
	s := helper.RegexpMatch(line, PhpFirstLineRegex)
	if len(s) > 0 {
		Date := helper.FormatTimeStamp(s[3], "")
		if Cf.Recover.From != "" && Date < helper.FormatTimeStamp(Cf.Recover.From, "") {
			return
		}

		if Cf.Recover.To != "" && Date > helper.FormatTimeStamp(Cf.Recover.To, "") {
			L.Debug("暂存过期，程序退出", LEVEL_NOTICE)
			cmd := exec.Command("/bin/bash", "-c", "cp "+helper.GetPathJoin(Cf.AppPath, "storage/data")+" /usr/local/postlog/storage/")
			_, err := cmd.Output()
			if err != nil {
				L.Debug(err.Error(), LEVEL_ERROR)
			}
			StopSignal <- os.Interrupt
			return
		}

		LogLine, _ := strconv.ParseInt(s[1], 10, 64)
		Xid := s[2]
		Remote := s[4]
		Country, Province, City, Jingwei := IP.GetLocation(Remote)
		Method := s[5]
		Url := strings.TrimSpace(s[6])
		Timestamp := fmt.Sprintf("%d", time.Now().Unix())
		u, err := ParseUrl(Url)
		Uri, DomainPort := "", ""
		if err != nil {
			L.Debug("url parse error"+err.Error()+",url:"+Url, LEVEL_ERROR)
		} else {
			//添加get参数
			if strings.Contains(PP.Rp.Pick, "get") {
				if len(u.Query()) > 0 {
					PP.setQuery(u.Query(), p)
				}
			}
			Uri = u.Path
			DomainPort = u.Host
		}
		WechatAppId, UserId, AppId := "", "", ""
		WechatMatchOld := helper.RegexpMatch(DomainPort, `^(wx\w+)\.`)
		if len(WechatMatchOld) > 0 {
			WechatAppId = WechatMatchOld[1]
		}
		WechatMatchDomain := helper.RegexpMatch(DomainPort, `^px-\w+-(\w+)-(\d+)-\w+\..*`)
		if len(WechatMatchDomain) > 0 {
			WechatAppId = WechatMatchDomain[1]
			UserId = WechatMatchDomain[2]
		}
		WechatMatchUri := helper.RegexpMatch(Uri, `^/api/wechat/mpapi/appid/(\w+)`)
		if len(WechatMatchUri) > 0 {
			WechatAppId = WechatMatchUri[1]
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
	}
}

func (PP *Processor) getMessage(p *object.PhpMsg, line string, index int, msgCh, nextCh chan int) {
	MatchMessage := helper.RegexpMatch(line, PhpMsgRegex)
	index -= 1
	if len(MatchMessage) > 0 {
		Message := object.Content{}
		Message.LogLine = MatchMessage[1]
		Message.Xid = MatchMessage[2]
		Message.LogType = MatchMessage[3]
		Message.Content = strings.TrimSpace(MatchMessage[4])
		if MatchMessage[3] == "error" {
			p.Tag = "php." + p.AppId + ".error"
			p.LogType = "error"
			if p.ErrorMsg == "" {
				p.ErrorMsg = Message.Content
			} else {
				p.ErrorMsg = p.ErrorMsg + "\n" + Message.Content
			}
		}
		p.Message[index] = Message

		//添加订单参数
		if strings.Contains(PP.Rp.Pick, "order") {
			PP.setMessageOrder(p, Message)
		}

		//添加请求参数
		if strings.Contains(PP.Rp.Pick, "params") {
			PP.setMessageParams(p, Message)
		}

		//添加header参数
		if strings.Contains(PP.Rp.Pick, "header") {
			PP.setMessageHeader(p, Message)
		}

		//添加runtime参数
		if strings.Contains(PP.Rp.Pick, "runtime") {
			PP.setMessageRuntime(p, Message)
		}

		//采集用户信息
		if p.UserId == "" && strings.Contains(PP.Rp.Pick, "user") {
			PP.setMessageUser(p, Message)
		}

		//采集微信回调信息
		if strings.Contains(PP.Rp.Pick, "wechat") {
			PP.setMessageWechat(p, Message)
		}
	}
	msgCh <- 0
	L.Debug(fmt.Sprintf("xid:%s,msgChlen:%d,len(msgCh):%d", p.Xid, cap(msgCh), len(msgCh)), LEVEL_DEBUG)
	if cap(msgCh) == len(msgCh) {
		nextCh <- 0
	}
}

//采集wechat参数
func (PP *Processor) setMessageWechat(p *object.PhpMsg, Message object.Content) {
	keywords := `[ WeChat ] [ MP ] [ API ] Message: `
	if strings.Contains(p.Uri, "/api/wechat/mpapi/appid/") && strings.Contains(Message.Content, keywords) {
		listContent := strings.Split(Message.Content, keywords)
		list := strings.Split(listContent[1], " [ FromPreLog:")
		wechatString := strings.Replace(list[0], `\"`, `"`, 100)
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

//采集耗时参数
func (PP *Processor) setMessageRuntime(p *object.PhpMsg, Message object.Content) {
	regexp := `^\[ (\w+) \] .*? \[ RunTime:(\d+\.\d+)s \]`
	res := helper.RegexpMatch(Message.Content, regexp)
	if len(res) > 0 {
		key := string(res[1])
		value, _ := strconv.ParseFloat(res[2], 64)
		value = helper.RoundFloat(value*1000, 2)
		runTimeLock.Lock()
		if exists, ok := p.RunTime[key]; ok {
			if exists < value {
				p.RunTime[key] = value
			}
		} else {
			p.RunTime[key] = value
		}
		runTimeLock.Unlock()
	}

	regexTotal := `^\[运行时间：(\d+.\d+)s\]\[吞吐率：(\d+.\d+)req/s\] \[内存消耗：((\d+,)?\d+.\d+)kb\] \[文件加载：(\d+)\]`
	resTotal := helper.RegexpMatch(Message.Content, regexTotal)
	if len(resTotal) > 0 {
		total, _ := strconv.ParseFloat(resTotal[1], 64)
		rpm, _ := strconv.ParseFloat(resTotal[2], 64)
		memory, _ := strconv.ParseFloat(strings.Replace(string(resTotal[3]), ",", "", 3), 32)
		runTimeLock.Lock()
		p.RunTime["total"] = helper.RoundFloat(total*1000, 2)
		p.RunTime["rpm"] = helper.RoundFloat(rpm, 2)
		p.RunTime["memory"] = helper.RoundFloat(memory, 2)
		include, _ := strconv.ParseFloat(resTotal[len(resTotal)-1], 64)
		p.RunTime["include"] = helper.RoundFloat(include, 2)
		runTimeLock.Unlock()
	}
}

//采集用户参数
func (PP *Processor) setMessageUser(p *object.PhpMsg, Message object.Content) {
	keywords := "get_db_connect"
	if len(Message.Content) >= len(keywords) && Message.Content[0:len(keywords)] == keywords {
		res := helper.RegexpMatch(Message.Content, `get_db_connect table:(\w+) params:(\d+)`)
		if len(res) > 0 {
			tableName := res[1]
			if strings.Contains(UserTable, tableName) {
				MsgLock.Lock()
				p.UserId = res[2]
				MsgLock.Unlock()
			}
		}
	}

	if len(Message.Content) >= 3 && Message.Content[0:3] == "OS:" {
		res := helper.RegexpMatch(Message.Content, PhpFrontCookie)
		if len(res) > 0 {
			MsgLock.Lock()
			if p.Country == "" {
				p.Country = res[2]
			}
			if p.Province == "" {
				p.Province = res[3]
			}
			if p.City == "" {
				p.City = res[4]
			}
			if res[6] != "" {
				p.UserId = res[6]
			}
			if res[7] != "" {
				p.OpenId = res[7]
			}
			p.ChannelId, p.AgentId, p.Operator, p.Access = res[8], res[9], res[5], res[1]
			MsgLock.Unlock()
		}
		res = helper.RegexpMatch(Message.Content, PhpAdminCookie)
		if len(res) > 0 {
			MsgLock.Lock()
			p.Country, p.Province, p.City, p.Operator, p.AdminId, p.Group = res[1], res[2], res[3], res[4], res[5], res[6]
			MsgLock.Unlock()
		}
	}
}

//采集订单参数
func (PP *Processor) setMessageOrder(p *object.PhpMsg, Message object.Content) {
	var res []string
	if strings.Contains(p.Uri, "/api/recharge/pay") || strings.Contains(p.Uri, "/api/activity/pay") {
		res = helper.RegexpMatch(Message.Content, PhpOrder)
		if len(res) > 0 {
			MsgLock.Lock()
			p.PayId, p.PayType, p.PayStatus, p.ChannelId, p.UserId, p.Money, p.GoodId = res[3], res[1], "create_"+res[2], res[4], res[5], helper.RoundString(res[6], 2), res[7]
			MsgLock.Unlock()
		}
	} else if strings.Contains(p.DomainPort, "callback") {
		res = helper.RegexpMatch(Message.Content, PhpOrderCallback)
		if len(res) > 0 {
			MsgLock.Lock()
			p.PayId, p.PayType, p.PayStatus, p.ChannelId, p.Money, p.GoodId = res[3], res[1], "callback_"+res[2], res[4], helper.RoundString(res[5], 2), res[6]
			MsgLock.Unlock()
		}
	}
}

func getQuery(v interface{}) string {
	r := reflect.TypeOf(v)
	if r.String() == "json.Number" {
		return v.(json.Number).String()
	} else if r.String() == "string" {
		return v.(string)
	}
	return ""
}

//采集请求参数
func (PP *Processor) setMessageParams(p *object.PhpMsg, Message object.Content) {
	if strings.Contains(Message.Content, "[ PARAM ] ") {
		list := strings.Split(Message.Content, `[ PARAM ] `)
		if len(list) == 2 && strings.HasPrefix(list[1], "{") {
			param, _ := simplejson.NewJson([]byte(list[1]))
			m, _ := param.Map()
			for k, v := range m {
				value := getQuery(v)
				if k == "code" && p.Uri == "/clientappapi" {
					runTimeLock.Lock()
					p.RequestTag["app_api_code"] = value
					runTimeLock.Unlock()
				}
				p.Request = append(p.Request, object.Query{Key: k, Value: value})
			}
		}
	}
}

//采集header参数
func (PP *Processor) setMessageHeader(p *object.PhpMsg, Message object.Content) {
	if strings.Contains(Message.Content, "[ HEADER ] ") {
		list := strings.Split(Message.Content, `[ HEADER ] `)
		if len(list) == 2 && strings.HasPrefix(list[1], "{") {
			param, _ := simplejson.NewJson([]byte(list[1]))
			m, _ := param.Map()
			for k, v := range m {
				value := getQuery(v)
				p.Header = append(p.Header, object.Query{Key: k, Value: value})
				if k == "common" {
					value, _ := simplejson.NewJson([]byte(getQuery(v)))
					uid, _ := value.Get("uid").String()
					if uid != "" {
						p.UserId = uid
					}
					token, _ := value.Get("token").String()
					if token != "" {
						p.OpenId = token
					}
				}

				if k == "cookie" && strings.Contains(PP.Rp.Pick, "cookie") {
					listCookie := strings.Split(value, "; ")
					for _, value := range listCookie {
						item := strings.Split(value, "=")
						if len(item) > 1 {
							p.Cookie = append(p.Cookie, object.Query{Key: item[0], Value: item[1]})
						}
					}
				}

				if k == "common" && strings.Contains(PP.Rp.Pick, "common") {
					common, _ := simplejson.NewJson([]byte(value))
					m, _ := common.Map()
					for key, value := range m {
						p.Common = append(p.Common, object.Query{Key: key, Value: getQuery(value)})
					}
				}
			}
		}
	}
}

//采集GET参数
func (PP *Processor) setQuery(values url.Values, msg *object.PhpMsg) {
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
		case "sid":
			msg.ChapterId = list[0]
			break
		case "agent_id":
			msg.AgentId = list[0]
			break
		case "user_id":
			msg.UserId = list[0]
			break
		}
	}
}

//获取主机id
func GetAppIdFromHostName(HostName string) string {
	if Cf.AppId != "" {
		return Cf.AppId
	} else {
		match := helper.RegexpMatch(HostName, `[[:alpha:]]+`)
		if len(match) > 0 {
			return match[0]
		} else {
			return HostName
		}
	}
}

//获取日志读取位置文件
func GetPositionFile(logType string) string {
	return helper.GetPathJoin(Cf.AppPath, ".position_"+logType)
}

//设置php读取行
func (PP *Processor) SetPhpLineNumber(line int64) int64 {
	PP.phpLineNumber = line
	return PP.phpLineNumber
}

//提高php读取行
func (PP *Processor) IncreaseLineNumber() int64 {
	PP.phpLineNumber++
	return PP.phpLineNumber
}

//设置php读取行
func (PP *Processor) SetPhpPostLineNumber(line int64) int64 {
	phpPostLineLock.Lock()
	defer phpPostLineLock.Unlock()
	if line > PP.phpPostLineNumber {
		PP.phpPostLineNumber = line
	}

	return PP.phpPostLineNumber
}

//获取php读取行
func (PP *Processor) GetPhpPostLineNumber() int64 {
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
