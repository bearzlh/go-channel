package service

import (
	"encoding/json"
	"github.com/bitly/go-simplejson"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"
	"workerChannel/object"
)

type EsService struct {
}

var Es *EsService

var BuckDoc = make(chan object.Doc, 10000)
var PostDoc = make(chan object.MsgInterface, 10000)
var BuckFull = make(chan bool, 1)
var BuckClose = make(chan bool, 1)
var EsCanUse = make(chan bool, 1)

func (E *EsService) CheckEsCanAccess() {
	go func() {
		t := time.NewTimer(time.Second * 1)
		for {
			select {
			case <-t.C:
				t.Reset(time.Second * 1)
				_, err := E.GetData("http://" + Cf.Es.Host)
				if err == nil {
					if len(EsCanUse) == 1 {
						L.Debug("es api recover", LEVEL_DEBUG)
						<-EsCanUse
					}
				} else {
					if len(EsCanUse) == 0 {
						L.Debug("es api error"+err.Error(), LEVEL_DEBUG)
						EsCanUse <- true
					}
				}
			}
		}
	}()
}

func (E *EsService) BuckWatch() {
	go func() {
		t := time.NewTimer(time.Second * time.Duration(Cf.Msg.BatchTimeSecond))
		for {
			select {
			case <-t.C:
				t.Reset(time.Second * time.Duration(Cf.Msg.BatchTimeSecond))
				if len(EsCanUse) == 0 {
					if len(BuckDoc) > 0 {
						L.Debug("timeout to post", LEVEL_DEBUG)
						Es.BuckPost(E.ProcessBulk())
					} else {
						L.Debug("timeout to post, nodata", LEVEL_DEBUG)
					}
				} else {
					if len(BuckDoc) > 0 {
						L.Debug("timeout to post, waiting", LEVEL_DEBUG)
					} else {
						L.Debug("timeout to post, nodata", LEVEL_DEBUG)
					}
				}
			case <-BuckClose:
				L.Debug("config changed, BuckClosed", LEVEL_DEBUG)
				return
			case <-BuckFull:
				if len(EsCanUse) == 0 {
					if len(BuckDoc) > 0 {
						L.Debug("size over to post", LEVEL_DEBUG)
						Es.BuckPost(E.ProcessBulk())
					}
				} else {
					if len(BuckDoc) > 0 {
						L.Debug("size over to post, wait", LEVEL_DEBUG)
						Es.BuckPost(E.ProcessBulk())
					}
				}

			}
		}
	}()
}

//添加数据
func (E *EsService) BuckAdd(msg object.MsgInterface) {
	BuckDoc <- object.Doc{Index: msg.GetIndexObj(msg.GetTimestamp()), Content: msg}
	if len(BuckDoc) > Cf.Msg.BatchSize {
		BuckFull <- true
	}
}

//组装批量数据
func (E *EsService) ProcessBulk() []string {
	bulkContent := make([]string, 0)
	lenBulk := Cf.Msg.BatchSize
	if len(BuckDoc) < Cf.Msg.BatchSize {
		lenBulk = len(BuckDoc)
	}
	for i := 0; i < lenBulk; i++ {
		doc := <-BuckDoc
		if doc.Content.GetName() == "php" {
			SetPhpPostLineNumber(doc.Content.GetLogLine(), false)
		} else {
			SetNginxPostLineNumber(doc.Content.GetLogLine(), false)
		}
		indexContent, _ := json.Marshal(doc.Index)
		Content, _ := json.Marshal(doc.Content)
		bulkContent = append(bulkContent, string(indexContent), string(Content))
		An.TimePostEnd = doc.Content.GetTimestamp()
	}
	return bulkContent
}

//发送失败重新放回发送队列
func (E *EsService) SaveDocToBulk(content []string) {
	for index := 0; index < len(content); index += 2 {
		BD := object.Doc{}
		err := json.Unmarshal([]byte(content[index]), &BD.Index)
		if err != nil {
			L.Debug("index process err"+err.Error(), LEVEL_ERROR)
			continue
		}
		if strings.Contains(BD.Index.IndexName.Index, "php") {
			msg := object.PhpMsg{}
			err1 := json.Unmarshal([]byte(content[index+1]), &msg)
			if err1 != nil {
				L.Debug("index process err"+err1.Error(), LEVEL_ERROR)
				continue
			}
			BD.Content = msg
		} else {
			msg := object.NginxMsg{}
			err1 := json.Unmarshal([]byte(content[index+1]), &msg)
			if err1 != nil {
				L.Debug("content process err"+err1.Error(), LEVEL_ERROR)
				continue
			}
			BD.Content = msg
		}
		BuckDoc <- BD
	}
}

//发送批量数据
func (E *EsService) BuckPost(content []string) bool {
	postData := strings.Join(content, "\n") + "\n"
	url := "http://"+Cf.Es.Host+"/_bulk"
	str, err := E.PostData(url, postData)
	jsonData, _ := simplejson.NewJson([]byte(str))
	errors, err := jsonData.Get("data").Get("errors").Bool()
	if errors {
		E.SaveDocToBulk(content)
		L.Debug(string(str)+err.Error(), LEVEL_ERROR)
		return errors
	} else {
		Lock.Lock()
		An.JobSuccess += int64(len(content) / 2)
		An.TimeEnd = time.Now().Unix()
		Lock.Unlock()
		L.Debug("发送成功", LEVEL_DEBUG)
		return false
	}
}

func (E *EsService) PostAdd(msg object.MsgInterface) {
	PostDoc <- msg
}

//单条数据发送
func (E *EsService) Post() {
	go func() {
		for {
			if len(EsCanUse) == 1 {
				L.Debug("es cannot use, wait", LEVEL_DEBUG)
				time.Sleep(time.Second * 5)
				continue
			}
			select {
			case msg := <-PostDoc:
				content, _ := json.Marshal(msg)
				url := "http://" + Cf.Es.Host + "/" + msg.GetIndex(msg.GetTimestamp())
				data := string(content)
				str, err := E.PostData(url, data)
				jsonData, _ := simplejson.NewJson([]byte(str))
				success, err := jsonData.Get("_shards").Get("successful").Int()
				if success == 0 {
					E.PostAdd(msg)
					L.Debug(string(str)+err.Error(), LEVEL_ERROR)
				} else {
					Lock.Lock()
					An.JobSuccess += 1
					An.TimeEnd = time.Now().Unix()
					Lock.Unlock()
				}
			}
		}
	}()
}

func (E *EsService) PostData(url string, content string) (string, error) {
	transport := http.Transport{
		Dial:              dialTimeout,
		DisableKeepAlives: true,
	}
	client := http.Client{
		Transport: &transport,
	}
	res, err := client.Post(url, "application/json", strings.NewReader(string(content)))
	if err != nil {
		return "", err
	}
	byteStr, err := ioutil.ReadAll(res.Body)
	return string(byteStr), err
}

func (E *EsService) GetData(url string) (string, error) {
	transport := http.Transport{
		Dial:              dialTimeout,
		DisableKeepAlives: true,
	}
	client := http.Client{
		Transport: &transport,
	}
	res, err := client.Get(url)
	if err != nil {
		return "", err
	}
	byteStr, err := ioutil.ReadAll(res.Body)
	return string(byteStr), err
}

func dialTimeout(network, addr string) (net.Conn, error) {
	conn, err := net.DialTimeout(network, addr, time.Second * 5)
	if err != nil {
		return conn, err
	}

	tcpConn := conn.(*net.TCPConn)
	err1 := tcpConn.SetKeepAlive(false)
	if err1 != nil {
		L.Debug("设置SetKeepAlive失败"+err1.Error(), LEVEL_ERROR)
	}

	return tcpConn, err1
}