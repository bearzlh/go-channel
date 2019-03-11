package object

import (
	"workerChannel/helper"
)

type PhpMsg struct {
	AppId       string `json:"appid"`
	WechatAppId string `json:"wechatappid"`
	HostName    string `json:"hostname"`
	Xid         string `json:"xid"`
	Date        int64  `json:"date"`
	Remote      string `json:"remote"`
	Method      string `json:"method"`
	Url         string `json:"url"`
	Uri         string `json:"uri"`
	DomainPort  string `json:"domainport"`
	Tag         string `json:"tag"`
	LogType     string `json:"log_type"`
	Timestamp   string `json:"timestamp"`

	LogLine     int64 `json:"log_line"`

	//query参数
	Query         []Query `json:"query"`
	BookId        string  `json:"book_id"`
	ReferralId    string  `json:"referral_id"`
	AgentId       string  `json:"agent_id"`
	ChapterId     string  `json:"chapter_id"`
	BookChapterId string  `json:"book_chapter_id"`

	//日志内容
	Message []Content `json:"message"`
}

type Query struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Content struct {
	Xid     string `json:"xid"`
	LogType string `json:"logtype"`
	Content string `json:"content"`
	LogLine string `json:"log_line"`
}

func (p PhpMsg) GetTimestamp() int64 {
	return p.Date
}

func (p PhpMsg) GetName() string {
	return "php"
}

func (p PhpMsg) GetPickTime() string {
	return p.Timestamp
}

func (p PhpMsg) GetLogType() string {
	return p.LogType
}

//获取当前索引对象
func (p PhpMsg) GetIndexObj(time int64) Index {
	return Index{IndexName: IndexContent{"log-php-" + helper.TimeFormat("Ymd", time), "go"}}
}

//获取索引
func (p PhpMsg) GetIndex(time int64) string {
	return "log-php-" + helper.TimeFormat("Ymd", time) + "/go"
}

func (p PhpMsg) GetLogLine() int64 {
	return p.LogLine
}