package object

type NginxMsg struct {
	AppId    string `json:"appid"`
	HostName string `json:"hostname"`
	Date     int64  `json:"date"`
	Remote   string `json:"remote"`
	Method   string `json:"method"`
	Url      string `json:"url"`
	Uri      string `json:"uri"`
	//DomainPort string `json:"domainport"`
	Tag        string  `json:"tag"`
	LogType    string  `json:"log_type"`
	Timestamp  string  `json:"timestamp"`
	HttpCode   string  `json:"http_code"`
	Referral   string  `json:"referral"`
	Browser    string  `json:"browser"`
	ServerPort string  `json:"server_port"`
	Cost       float64 `json:"cost"`
	UpCost     float64 `json:"up_cost"`
	SentBytes  int64   `json:"sent_b"`
	XForward   string  `json:"x_forward"`
	UpStream   string  `json:"up_stream"`

	LogLine int64 `json:"log_line"`

	//query参数
	Query []Query `json:"query"`
}

func (Ng NginxMsg) GetName() string {
	return "nginx"
}

func (Ng NginxMsg) GetLogLine() int64 {
	return Ng.LogLine
}

func (Ng NginxMsg)GetTimestamp() int64 {
	return Ng.Date
}

func (Ng NginxMsg)GetPickTime() string {
	return Ng.Timestamp
}

func (Ng NginxMsg)GetLogType() string {
	return Ng.LogType
}

func (Ng NginxMsg) GetIndex(env string, format string, time int64) string {
	return GetIndex(env, format, time, "nginx") + "/go"
}

func (Ng NginxMsg) GetIndexObj(env string, format string, time int64) Index {
	return Index{IndexName: IndexContent{GetIndex(env, format, time, "nginx"), "go"}}
}

func (Ng NginxMsg)GetJobId() string {
	return Ng.Timestamp
}