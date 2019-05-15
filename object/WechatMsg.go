package object

type WechatMsg struct {
	MsgType  string `json:"MsgType"`
	Event    string `json:"Event"`
	EventKey string `json:"EventKey"`
	Content  string `json:"Content"`
}
