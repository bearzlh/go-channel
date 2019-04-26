package service

import (
	"encoding/json"
	"fmt"
	"github.com/bitly/go-simplejson"
	"io/ioutil"
	"math/rand"
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
				_, err := E.GetData("http://" + E.GetHost())
				if err == nil {
					if len(EsCanUse) == 1 {
						L.Debug("es api recover", LEVEL_NOTICE)
						<-EsCanUse
					}
				} else {
					if len(EsCanUse) == 0 {
						L.Debug("es api error"+err.Error(), LEVEL_ERROR)
						EsCanUse <- true
					}
				}
			}
		}
	}()
}

func (E *EsService) GetHost() string {
	hostStrings := strings.Split(Cf.Es.Host, ",")
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)
	return hostStrings[r.Intn(len(hostStrings))]
}

func (E *EsService) BuckWatch() {
	go func() {
		t := time.NewTimer(time.Second * time.Duration(Cf.Msg.BatchTimeSecond))
		for {
			select {
			case <-BuckClose:
				L.Debug("config changed, BuckClosed", LEVEL_NOTICE)
				break
			case <-t.C:
				t.Reset(time.Second * time.Duration(Cf.Msg.BatchTimeSecond))
				if len(EsCanUse) == 0 {
					if len(BuckDoc) > 0 {
						L.Debug("timeout to post", LEVEL_NOTICE)
						phpLine, content := E.ProcessBulk()
						go func() {
							Es.BuckPost(phpLine, content)
						}()
					} else {
						L.Debug("timeout to post, nodata", LEVEL_DEBUG)
					}
				} else {
					if len(BuckDoc) > 0 {
						L.Debug("timeout to post, waiting", LEVEL_NOTICE)
					} else {
						L.Debug("timeout to post, nodata", LEVEL_DEBUG)
					}
				}
			case <-BuckFull:
				t.Reset(time.Second * time.Duration(Cf.Msg.BatchTimeSecond))
				if len(BuckDoc) > 0 {
					L.Debug("size over to post", LEVEL_NOTICE)
					phpLine, content := E.ProcessBulk()
					go func() {
						Es.BuckPost(phpLine, content)
					}()
				}

			}
		}
	}()
}

//添加数据
func (E *EsService) BuckAdd(msg object.MsgInterface) {
	BuckDoc <- object.Doc{Index: msg.GetIndexObj(Cf.Env, Cf.Es.IndexFormat, msg.GetTimestamp()), Content: msg}
	if len(BuckDoc) > Cf.Msg.BatchSize && len(EsCanUse) == 0 {
		BuckFull <- true
	}
}

//组装批量数据
func (E *EsService) ProcessBulk() (int64, []string) {
	bulkContent := make([]string, 0)
	lenBulk := Cf.Msg.BatchSize
	if len(BuckDoc) < Cf.Msg.BatchSize {
		lenBulk = len(BuckDoc)
	}
	phpLine := int64(0)
	for i := 0; i < lenBulk; i++ {
		doc := <-BuckDoc
		if doc.Content.GetLogLine() > phpLine {
			phpLine = doc.Content.GetLogLine()
		}
		indexContent, _ := json.Marshal(doc.Index)
		Content, _ := json.Marshal(doc.Content)
		bulkContent = append(bulkContent, string(indexContent), string(Content))
		An.TimePostEnd = doc.Content.GetTimestamp()
	}
	L.Debug(fmt.Sprintf("组装数据,%d,%d", phpLine, len(bulkContent)), LEVEL_INFO)
	return phpLine, bulkContent
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
func (E *EsService) BuckPost(phpLine int64, content []string) bool {
	if !Cf.Factory.On {
		L.Debug(fmt.Sprintf("暂停数据发送,%d", phpLine), LEVEL_INFO)
		return false
	}
	postData := strings.Join(content, "\n") + "\n"
	url := "http://"+E.GetHost()+"/_bulk"
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
		SetPhpPostLineNumber(phpLine, false)
		L.Debug(fmt.Sprintf("发送成功,%d", phpLine), LEVEL_INFO)
		L.Debug(postData, LEVEL_DEBUG)
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
				L.Debug("es cannot use, wait", LEVEL_INFO)
				time.Sleep(time.Second * 5)
				continue
			}
			select {
			case msg := <-PostDoc:
				content, _ := json.Marshal(msg)
				url := "http://" + E.GetHost() + "/" + msg.GetIndex(Cf.Env, Cf.Es.IndexFormat, msg.GetTimestamp())
				data := string(content)
				L.Debug(url+data, LEVEL_DEBUG)
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