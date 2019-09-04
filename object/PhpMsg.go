package object

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
	ChapterId     string  `json:"chapter_id"`

	//Cookie参数
	UserId    string `json:"user_id"`
	OpenId    string `json:"open_id"`
	ChannelId string `json:"channel_id"`
	AgentId   string `json:"agent_id"`
	AdminId   string `json:"admin_id"`
	Group     string `json:"group"`

	//地域参数
	Country  string `json:"country"`
	Province string `json:"province"`
	City     string `json:"city"`
	Operator string `json:"operator"`
	Access   string `json:"access"`

	//订单参数
	PayId     string  `json:"pay_id"`
	PayType   string  `json:"pay_type"`   //wechatpay,mihua...
	PayStatus string  `json:"pay_status"` //create_success,create_fail,callback
	GoodId    string  `json:"good_id"`
	Money     float64 `json:"money"`

	Jingwei string `json:"jingwei"`

	RequestTag map[string]string `json:"request_tag"`

	RunTime map[string]float64 `json:"runtime"`

	//日志内容
	Message []Content `json:"message"`

	ErrorMsg string `json:"error_msg"`

	Request []Query `json:"request"`

	Header []Query `json:"header"`

	Cookie []Query `json:"cookie"`

	Common []Query `json:"common"`

	WechatMsg *WechatMsg `json:"wechat_msg"`
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
func (p PhpMsg) GetIndexObj(env string, format string, time int64) Index {
	return Index{IndexName: IndexContent{GetIndex(env, format, time, "php"), "go"}}
}

//获取索引
func (p PhpMsg) GetIndex(env string, format string, time int64) string {
	index := GetIndex(env, format, time, "php")
	return index + "/go"
}

func (p PhpMsg)GetJobId() string {
	return p.Xid
}