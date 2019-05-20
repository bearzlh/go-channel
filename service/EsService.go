package service

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/bitly/go-simplejson"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
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
var EsRunning = int64(0)

func (E *EsService) Init() {
	EsRunning = time.Now().Unix()
	if Cf.Recover.From == "" {
		//es是否可用
		E.CheckEsCanAccess()
		//单条数据发送
		E.Post()
		//检测暂存
		ConcurrentPost = make(chan int, Cf.Es.ConcurrentPost)
		ThreadLimit = make(chan int, Cf.Es.RecoverThread)
		E.CheckStorage()
	}
	//检测批量发送队列
	Es.BuckWatch()
}

//检查es是否可用
func (E *EsService) CheckEsCanAccess() {
	go func() {
		t := time.NewTimer(time.Second * 2)
		for {
			select {
			case <-t.C:
				t.Reset(time.Second * 2)
				//indexName := object.GetIndex(Cf.Env, Cf.Es.IndexFormat, time.Now().Unix(), "php")
				//url := "http://"+E.GetHost()+"/"+indexName+"/_settings"
				//_, err := E.GetData(url)
				allowIndex := AllowIndex()
				if allowIndex {
					if len(EsCanUse) == 1 {
						L.Debug("es api recover", LEVEL_NOTICE)
						<-EsCanUse
					}
					if len(Storage) == 0 {
						Storage <- true
					}
				} else {
					if len(EsCanUse) == 0 {
						L.Debug("es api error", LEVEL_ERROR)
						EsCanUse <- true
					}
				}
			}
		}
	}()
}

//检查是否可以进行索引
func AllowIndex() bool {
	_, err := Es.GetData("http://" + Es.GetHost())
	//接口不通
	if err != nil {
		L.Debug("es不可用", LEVEL_ERROR)
		return false
	}

	indexName := object.GetIndex(Cf.Env, Cf.Es.IndexFormat, time.Now().Unix(), "php")
	url := "http://" + Es.GetHost() + "/" + indexName + "/_settings"
	content, err := Es.GetData(url)
	if err != nil {
		L.Debug("es不可用,content:"+err.Error(), LEVEL_ERROR)
		return false
	}
	jsonContent, err := simplejson.NewJson([]byte(content))
	if err != nil {
		L.Debug("es不可用,jsonContent:"+err.Error(), LEVEL_ERROR)
		return false
	}
	blocks, err := jsonContent.GetPath(indexName, "settings", "index", "blocks").Map()
	if err != nil {
		return true
	}
	if data, ok := blocks["read_only_allow_delete"]; ok && data == "true" {
		//只可删除，不可索引
		return false
	} else {
		//可以进行索引
		return true
	}
}

//获取es地址
func (E *EsService) GetHost() string {
	hostStrings := strings.Split(Cf.Es.Host, ",")
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)
	return hostStrings[r.Intn(len(hostStrings))]
}

//检查发送队列
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
					L.Debug("time up to post", LEVEL_INFO)
					content, jobs := E.ProcessBulk()
					go func() {
						Es.BuckPost(content, jobs)
					}()
				} else {
					L.Debug("timeout to post, nodata", LEVEL_DEBUG)
				}
			case <-BuckFull:
				t.Reset(time.Second * time.Duration(Cf.Msg.BatchTimeSecond))
				if len(BuckDoc) > 0 {
					L.Debug("size over to post", LEVEL_INFO)
					content, jobs := E.ProcessBulk()
					go func() {
						Es.BuckPost(content, jobs)
					}()
				}

			}
		}
	}()
}

//检查暂存
func (E *EsService) CheckStorage() {
	go func() {
		for {
			if StopStatus {
				L.Debug("check storage stopped", LEVEL_NOTICE)
				return
			}
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
				count := int64(0)
				L.Debug(fmt.Sprintf("send loop start,backup line:%d", An.BackUpLine), LEVEL_INFO)
				sending := make(chan bool, 1)
				sendingLock := new(sync.Mutex)
				for {
					line, errRead := rd.ReadString('\n')
					if line != "" {
						count++
						Lock.Lock()
						if count < An.BackUpLine {
							Lock.Unlock()
							continue
						}
						Lock.Unlock()
						dataPost = append(dataPost, line)
					}
					if count%2 == 0 {
						if len(dataPost) >= Cf.Msg.BatchSize*2 {
							postData := strings.Join(dataPost, "")
							ThreadLimit<-1
							L.Debug(fmt.Sprintf("线程数, %d, %d", len(ThreadLimit), len(dataPost)), LEVEL_INFO)
							dataPost = make([]string, 0)
							go func() {
								sendingLock.Lock()
								if len(sending) == 0 {
									sending<-true
								}
								sendingLock.Unlock()
								_, err := E.PostData("http://"+E.GetHost()+"/_bulk", postData)
								if err != nil {
									L.Debug("暂存数据存储错误"+err.Error(), LEVEL_ERROR)
									E.SaveToStorage(postData)
								} else {
									Lock.Lock()
									An.BackUpLine = count
									Lock.Unlock()
								}
								sendingLock.Lock()
								if len(sending) != 0 {
									<-sending
								}
								sendingLock.Unlock()
								<-ThreadLimit
							}()
						}
						if errRead != nil || io.EOF == errRead {
							if len(dataPost) > 0 {
								postData := strings.Join(dataPost, "")
								ThreadLimit<-1
								L.Debug(fmt.Sprintf("线程数, %d, %d", len(ThreadLimit), len(dataPost)), LEVEL_INFO)
								go func() {
									sendingLock.Lock()
									if len(sending) == 0 {
										sending<-true
									}
									sendingLock.Unlock()
									_, err := E.PostData("http://"+E.GetHost()+"/_bulk", postData)
									if err != nil {
										L.Debug("暂存数据存储错误"+err.Error(), LEVEL_ERROR)
										E.SaveToStorage(postData)
									}
									sendingLock.Lock()
									if len(sending) != 0 {
										<-sending
									}
									sendingLock.Unlock()
									<-ThreadLimit
								}()
							}
							break
						}
					}
				}
				for {
					time.Sleep(time.Second * 5)
					if len(ThreadLimit) == 0 {
						L.Debug(fmt.Sprintf("send loop end, 发送数据量:%d", count/2), LEVEL_INFO)
						errClose := f.Close()
						if errClose != nil {
							L.Debug("文件关闭失败"+f.Name()+err.Error(), LEVEL_ERROR)
						}
						L.Debug("文件关闭"+f.Name(), LEVEL_INFO)
						errRemove := os.Remove(fileName)
						Lock.Lock()
						An.BackUpLine = 0
						Lock.Unlock()
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
						break
					}
				}
			}
		}
	}()
}

//将内存暂存到本地
func (E *EsService) SaveToStorage(content string) {
	dir := helper.GetPathJoin(Cf.AppPath, Cf.Es.Storage)
	if !helper.IsDir(dir) {
		err := helper.Mkdir(dir)
		if err != nil {
			L.Debug("暂存目录无法创建"+err.Error(), LEVEL_ERROR)
			return
		}
	}

	name := "data"
	if len(ThreadLimit) > 0 {
		name = "back"
	}
	fileName := helper.GetPathJoin(Cf.AppPath, Cf.Es.Storage, name)
	err := helper.FilePutContents(fileName, content, true)
	if err != nil {
		L.Debug("日志暂存失败"+err.Error(), LEVEL_ERROR)
	} else {
		L.Debug("暂存完成", LEVEL_INFO)
	}
}

//添加数据
func (E *EsService) BuckAdd(msg object.MsgInterface) {
	BuckDoc <- object.Doc{Index: msg.GetIndexObj(Cf.Env, Cf.Es.IndexFormat, msg.GetTimestamp()), Content: msg}
	if len(BuckDoc) > Cf.Msg.BatchSize {
		BuckFull <- true
	}
}

//组装批量数据
func (E *EsService) ProcessBulk() (string, string) {
	bulkContent := make([]string, 0)
	lenBulk := Cf.Msg.BatchSize
	if len(BuckDoc) < Cf.Msg.BatchSize {
		lenBulk = len(BuckDoc)
	}
	jobs := make([]string, lenBulk)
	for i := 0; i < lenBulk; i++ {
		doc := <-BuckDoc
		indexContent, _ := json.Marshal(doc.Index)
		Content, _ := json.Marshal(doc.Content)
		bulkContent = append(bulkContent, string(indexContent), string(Content))
		An.TimePostEnd = doc.Content.GetTimestamp()
		jobs[i] = doc.Content.GetJobId()
	}
	L.Debug(fmt.Sprintf("组装数据,%d", len(bulkContent)), LEVEL_INFO)
	return strings.Join(bulkContent, "\n") + "\n", strings.Join(jobs, ",")
}

//php数据暂存
func (E *EsService)PhpDataSave(content string)  {
	E.SaveToStorage(content)
}

//发送批量数据
func (E *EsService) BuckPost(content string, jobs string) bool {
	EsRunning = time.Now().Unix()
	Lock.Lock()
	An.TimeEnd = time.Now().Unix()
	Lock.Unlock()
	if Cf.Recover.From != "" {
		L.Debug("数据恢复中", LEVEL_INFO)
		E.PhpDataSave(content)
		return false
	}

	if !Cf.Factory.On {
		L.Debug("暂停数据发送", LEVEL_NOTICE)
		E.PhpDataSave(content)
		return false
	}
	if len(EsCanUse) > 0 {
		L.Debug("es不可用，进行暂存", LEVEL_INFO)
		E.PhpDataSave(content)
		return false
	}
	url := "http://"+E.GetHost()+"/_bulk"
	str, err := E.PostData(url, content)
	if err != nil {
		L.Debug("es发送错误，进行暂存", LEVEL_INFO)
		E.PhpDataSave(content)
		return false
	}
	jsonData, _ := simplejson.NewJson([]byte(str))
	errorsGet, err := jsonData.Get("data").Get("errors").Bool()
	if errorsGet {
		L.Debug("es返回值错误，进行暂存"+err.Error(), LEVEL_INFO)
		E.PhpDataSave(content)
		return errorsGet
	} else {
		Lock.Lock()
		An.JobSuccess += int64(len(content) / 2)
		Lock.Unlock()
		L.Debug("发送成功,jobs-->"+jobs, LEVEL_INFO)
		return true
	}
}

//单条数据入列
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
					E.SaveToStorage(data)
					continue
				}
				jsonData, _ := simplejson.NewJson([]byte(str))
				success, err := jsonData.Get("_shards").Get("successful").Int()
				if success == 0 {
					L.Debug("es返回值错误，进行暂存"+err.Error(), LEVEL_ERROR)
					E.SaveToStorage(data)
				}
			}
		}
	}()
}

//执行数据的发送
func (E *EsService) PostData(url string, content string) (string, error) {
	transport := http.Transport{
		DialContext: dialTimeout,
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
							L.Debug(fmt.Sprintf("返回值错误, code:%d, %s, %s", value.Index.Status, value.Index.Error.Type, value.Index.Error.Reason), LEVEL_ERROR)
							contentNew = append(contentNew, contentArr[index*2])
							contentNew = append(contentNew, contentArr[index*2+1])
						}
					}
					content = strings.Join(contentNew, "\n") + "\n"
					err = errors.New("es请求错误")
					continue
				} else {
					err = nil
					break
				}
			}
			break
		} else {
			L.Debug("post error"+string(byteC), LEVEL_ERROR)
			err = errors.New("es请求错误")
		}
	}
	if err == nil {
		L.Debug("post success", LEVEL_NOTICE)
	}
	return string(byteC), err
}

//执行get请求
func (E *EsService) GetData(url string) (string, error) {
	transport := http.Transport{
		DialContext: dialTimeout,
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

//api请求参数配置
func dialTimeout(ctx context.Context, network, addr string) (net.Conn, error) {
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