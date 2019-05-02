package service

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/bitly/go-simplejson"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"workerChannel/helper"
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
var Storage = make(chan bool, 1)
var ConcurrentPost chan int
var ThreadLimit chan int

func (E *EsService) Init() {
	if Cf.Recover.From == "" {
		//es是否可用
		E.CheckEsCanAccess()
		//单条数据发送
		E.Post()
		//检测暂存
		E.CheckStorage()
		ConcurrentPost = make(chan int, Cf.Es.ConcurrentPost)
		ThreadLimit = make(chan int, Cf.Es.RecoverThread)
	}
	//检测批量发送队列
	Es.BuckWatch()
}

func (E *EsService) CheckEsCanAccess() {
	go func() {
		t := time.NewTimer(time.Second * 2)
		for {
			select {
			case <-t.C:
				t.Reset(time.Second * 2)
				_, err := E.GetData("http://" + E.GetHost())
				if err == nil {
					if len(EsCanUse) == 1 {
						L.Debug("es api recover", LEVEL_NOTICE)
						<-EsCanUse
					}
					if len(Storage) == 0 {
						Storage <- true
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
				if len(BuckDoc) > 0 {
					L.Debug("time up to post", LEVEL_NOTICE)
					phpLine, content, jobs := E.ProcessBulk()
					go func() {
						Es.BuckPost(phpLine, content, jobs)
					}()
				} else {
					L.Debug("timeout to post, nodata", LEVEL_DEBUG)
				}
			case <-BuckFull:
				t.Reset(time.Second * time.Duration(Cf.Msg.BatchTimeSecond))
				if len(BuckDoc) > 0 {
					L.Debug("size over to post", LEVEL_NOTICE)
					phpLine, content, jobs := E.ProcessBulk()
					go func() {
						Es.BuckPost(phpLine, content, jobs)
					}()
				}

			}
		}
	}()
}

//检测暂存
func (E *EsService) CheckStorage() {
	go func() {
		for {
			select {
			case <-Storage:
				fileName := helper.GetPathJoin(Cf.AppPath, Cf.Es.Storage, "data")
				if !helper.IsFile(fileName) {
					continue
				}
				f, err := os.Open(fileName)
				if err != nil {
					L.Debug(fileName+err.Error(), LEVEL_INFO)
					continue
				}

				rd := bufio.NewReader(f)
				dataPost := make([]string, 0)
				count := 0
				L.Debug("send loop start", LEVEL_INFO)
				sending := make(chan bool, 1)
				sengindLock := new(sync.Mutex)
				for {
					line, errRead := rd.ReadString('\n')
					if line != "" {
						count++
						dataPost = append(dataPost, line)
					}
					if count%2 == 0 {
						if len(dataPost) >= Cf.Msg.BatchSize*2 {
							postData := strings.Join(dataPost, "")
							dataPost = make([]string, Cf.Msg.BatchSize*2)
							ThreadLimit<-1
							L.Debug(fmt.Sprintf("线程数, %d", len(ThreadLimit)), LEVEL_INFO)
							go func() {
								sengindLock.Lock()
								if len(sending) == 0 {
									sending<-true
								}
								sengindLock.Unlock()
								_, err := E.PostData("http://"+E.GetHost()+"/_bulk", postData)
								if err != nil {
									L.Debug("暂存数据存储错误"+err.Error(), LEVEL_ERROR)
									E.SaveToStorage(postData)
								}
								sengindLock.Lock()
								if len(sending) != 0 {
									<-sending
								}
								sengindLock.Unlock()
								<-ThreadLimit
							}()
						}
						if errRead != nil || io.EOF == errRead {
							if len(dataPost) > 0 {
								postData := strings.Join(dataPost, "")
								ThreadLimit<-1
								go func() {
									sengindLock.Lock()
									if len(sending) == 0 {
										sending<-true
									}
									sengindLock.Unlock()
									_, err := E.PostData("http://"+E.GetHost()+"/_bulk", postData)
									if err != nil {
										L.Debug("暂存数据存储错误"+err.Error(), LEVEL_ERROR)
										E.SaveToStorage(postData)
									}
									sengindLock.Lock()
									if len(sending) != 0 {
										<-sending
									}
									sengindLock.Unlock()
									<-ThreadLimit
								}()
							}
							break
						}
					}
				}
				L.Debug(fmt.Sprintf("send loop end, 发送数据量:%d", count/2), LEVEL_INFO)
				errClose := f.Close()
				if errClose != nil {
					L.Debug("文件关闭失败"+f.Name()+err.Error(), LEVEL_ERROR)
				}
				L.Debug("文件关闭"+f.Name(), LEVEL_INFO)
				errRemove := os.Remove(fileName)
				if errRemove != nil {
					L.Debug("文件删除失败"+fileName, LEVEL_ERROR)
				}
				L.Debug("文件删除"+f.Name(), LEVEL_INFO)
				back := helper.GetPathJoin(Cf.AppPath, Cf.Es.Storage, "back")
				if helper.IsFile(back) {
					errRename := os.Rename(back, fileName)
					if errRename != nil {
						L.Debug("文件重命名失败,from:"+back+" to:"+fileName, LEVEL_ERROR)
					} else {
						L.Debug("文件重命名,from:"+back+" to:"+fileName, LEVEL_INFO)
					}
				}
			}
		}
	}()
}

func (E *EsService)SaveToStorage(content string) {
	dir := helper.GetPathJoin(Cf.AppPath, Cf.Es.Storage)
	if !helper.IsDir(dir) {
		err := helper.Mkdir(dir)
		if err != nil {
			L.Debug("目录无法创建"+err.Error(), LEVEL_ALERT)
			return
		}
	}

	name := "data"
	if len(ThreadLimit) > 0 {
		name = "back"
	}
	fileName := helper.GetPathJoin(Cf.AppPath, Cf.Es.Storage, name)
	L.Debug("数据暂存"+name, LEVEL_INFO)
	L.WriteAppend(fileName, content)
}

//添加数据
func (E *EsService) BuckAdd(msg object.MsgInterface) {
	BuckDoc <- object.Doc{Index: msg.GetIndexObj(Cf.Env, Cf.Es.IndexFormat, msg.GetTimestamp()), Content: msg}
	if len(BuckDoc) > Cf.Msg.BatchSize {
		BuckFull <- true
	}
}

//组装批量数据
func (E *EsService) ProcessBulk() (int64, string, string) {
	bulkContent := make([]string, 0)
	lenBulk := Cf.Msg.BatchSize
	if len(BuckDoc) < Cf.Msg.BatchSize {
		lenBulk = len(BuckDoc)
	}
	phpLine := int64(0)
	jobs := make([]string, lenBulk)
	for i := 0; i < lenBulk; i++ {
		doc := <-BuckDoc
		if doc.Content.GetLogLine() > phpLine {
			phpLine = doc.Content.GetLogLine()
		}
		indexContent, _ := json.Marshal(doc.Index)
		Content, _ := json.Marshal(doc.Content)
		bulkContent = append(bulkContent, string(indexContent), string(Content))
		An.TimePostEnd = doc.Content.GetTimestamp()
		jobs[i] = doc.Content.GetJobId()
	}
	L.Debug(fmt.Sprintf("组装数据,%d,%d", phpLine, len(bulkContent)), LEVEL_INFO)
	return phpLine, strings.Join(bulkContent, "\n") + "\n", strings.Join(jobs, ",")
}

//php数据暂存
func (E *EsService)PhpDataSave(phpLine int64, content string)  {
	E.SaveToStorage(content)
	Lock.Lock()
	An.TimeEnd = time.Now().Unix()
	Lock.Unlock()
	SetPhpPostLineNumber(phpLine, false)
}

//发送批量数据
func (E *EsService) BuckPost(phpLine int64, content string, jobs string) bool {
	if Cf.Recover.From != "" {
		L.Debug(fmt.Sprintf("数据恢复中,%d", phpLine), LEVEL_INFO)
		E.PhpDataSave(phpLine, content)
		return false
	}

	if !Cf.Factory.On {
		L.Debug(fmt.Sprintf("暂停数据发送,%d", phpLine), LEVEL_INFO)
		E.PhpDataSave(phpLine, content)
		return false
	}
	if len(EsCanUse) > 0 {
		L.Debug(fmt.Sprintf("es不可用，进行暂存，%d", phpLine), LEVEL_INFO)
		E.PhpDataSave(phpLine, content)
		return false
	}
	url := "http://"+E.GetHost()+"/_bulk"
	str, err := E.PostData(url, content)
	if err != nil {
		L.Debug(fmt.Sprintf("es发送错误，进行暂存，%d", phpLine), LEVEL_INFO)
		E.PhpDataSave(phpLine, content)
		return false
	}
	jsonData, _ := simplejson.NewJson([]byte(str))
	errors, err := jsonData.Get("data").Get("errors").Bool()
	if errors {
		L.Debug(fmt.Sprintf("es返回值错误，进行暂存，%d"+err.Error(), phpLine), LEVEL_INFO)
		E.PhpDataSave(phpLine, content)
		return errors
	} else {
		Lock.Lock()
		An.JobSuccess += int64(len(content) / 2)
		An.TimeEnd = time.Now().Unix()
		Lock.Unlock()
		SetPhpPostLineNumber(phpLine, false)
		L.Debug(fmt.Sprintf("发送成功,%d,jobs-->"+jobs, phpLine), LEVEL_INFO)
		L.Debug(content, LEVEL_DEBUG)
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
			select {
			case msg := <-PostDoc:
				content, _ := json.Marshal(msg)
				index := msg.GetIndexObj(Cf.Env, Cf.Es.IndexFormat, msg.GetTimestamp())
				url := "http://" + E.GetHost() + "/" + index.IndexName.Index + "/" + index.IndexName.Type
				data := string(content)
				if len(EsCanUse) == 1 {
					L.Debug("es cannot use, wait", LEVEL_INFO)
					indexContent, _ := json.Marshal(index)
					E.SaveToStorage(string(indexContent) + "\n" + data + "\n")
					continue
				}
				L.Debug(url+data, LEVEL_DEBUG)
				str, err := E.PostData(url, data)
				if err != nil {
					L.Debug("es发送错误，进行暂存"+err.Error(), LEVEL_ERROR)
					E.PostAdd(msg)
					continue
				}
				jsonData, _ := simplejson.NewJson([]byte(str))
				success, err := jsonData.Get("_shards").Get("successful").Int()
				if success == 0 {
					L.Debug("es返回值错误，进行暂存"+err.Error(), LEVEL_ERROR)
					E.PostAdd(msg)
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
	var res *http.Response
	var err error
	var byteC []byte
	for i := 0; i < Cf.Es.Retry; i++ {
		ConcurrentPost<-0
		res, err = client.Post(url, "application/json", strings.NewReader(string(content)))
		<-ConcurrentPost
		if err != nil {
			L.Debug("es请求错误，"+err.Error(), LEVEL_ERROR)
			break
		} else {
			L.Debug("es请求成功", LEVEL_INFO)
		}
		byteC, _ = ioutil.ReadAll(res.Body)
		if res.StatusCode == 200 || res.StatusCode == 201 {
			if strings.Contains(url, "_bulk") {
				buckRes := new(object.EsBuckResponse)
				errJson := json.Unmarshal(byteC, buckRes)
				if errJson != nil {
					L.Debug("json格式化错误"+errJson.Error(), LEVEL_ERROR)
					break
				}
				if buckRes.Errors {
					contentArr := strings.Split(content, "\n")
					contentNew := make([]string, 0)
					for index, value := range buckRes.Items {
						if value.Index.Status != 201 {
							L.Debug(fmt.Sprintf("返回值错误, code:%d, %s, %s", value.Index.Status, value.Index.Error.Type, value.Index.Error.Reason), LEVEL_NOTICE)
							contentNew = append(contentNew, contentArr[index*2])
							contentNew = append(contentNew, contentArr[index*2+1])
						}
					}
					content = strings.Join(contentNew, "\n") + "\n"
					continue
				}
			}
			break
		} else {
			L.Debug("post error"+string(byteC), LEVEL_ERROR)
		}
	}
	if err == nil {
		L.Debug("post success", LEVEL_NOTICE)
	}
	return string(byteC), err
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