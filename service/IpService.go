package service

import (
	"fmt"
	"github.com/oschwald/geoip2-golang"
	"net"
	"sync"
	"time"
	"workerChannel/helper"
)

type IpService struct {
	DB *geoip2.Reader
}

var IP IpService

var IPCache = map[string][4]string{}
var IPTime = map[string]int64{}
var lock = new(sync.Mutex)

//初始化
func (I *IpService) GetDB() {
	lock.Lock()
	if I.DB == nil {
		DB, err := geoip2.Open(helper.GetPathJoin(Cf.AppPath, "./GeoLite2-City.mmdb"))
		if err != nil {
			L.Debug("error to open mmdb", LEVEL_NOTICE)
			return
		}
		I.CheckCache()
		I.DB = DB
	}
	lock.Unlock()
}

//获取地域信息
func (I *IpService) GetLocation(ip string) (string, string, string, string) {
	var guo, sheng, shi, jingwei = "", "", "", ""
	if I.DB == nil {
		I.GetDB()
	}
	if I.DB == nil {
		return guo, sheng, shi, jingwei
	}
	lock.Lock()
	_, ok := IPCache[ip]
	if ok {
		IPTime[ip] = time.Now().Unix()
		guo, sheng, shi, jingwei = IPCache[ip][0], IPCache[ip][1], IPCache[ip][2], IPCache[ip][3]
		lock.Unlock()
	} else {
		lock.Unlock()
		// If you are using strings that may be invalid, check that ip is not nil
		ipParam := net.ParseIP(ip)
		record, err := I.DB.City(ipParam)
		if err != nil {
			L.Debug("error to parse ip"+ip, LEVEL_NOTICE)
			lock.Lock()
			IPTime[ip] = time.Now().Unix()
			IPCache[ip] = [4]string{guo, sheng, shi, jingwei}
			lock.Unlock()
		} else {
			guo = record.Country.Names["zh-CN"]
			if len(record.Subdivisions) > 0 {
				sheng = record.Subdivisions[0].Names["zh-CN"]
			}
			shi = record.City.Names["zh-CN"]
			if record.Location.Latitude != 0 && record.Location.Longitude != 0 {
				jingwei = fmt.Sprintf("%.4f,%.4f", record.Location.Latitude, record.Location.Longitude)
			}
			lock.Lock()
			IPTime[ip] = time.Now().Unix()
			IPCache[ip] = [4]string{guo, sheng, shi, jingwei}
			lock.Unlock()
		}
	}

	return guo, sheng, shi, jingwei
}

//检查缓存
func (I *IpService) CheckCache() {
	go func() {
		L.Debug("check ip cache", LEVEL_NOTICE)
		checkTime := time.NewTimer(time.Second * time.Duration(Cf.Msg.IpCheckInterval))
		for {
			select {
			case <-checkTime.C:
				checkTime.Reset(time.Second * time.Duration(Cf.Msg.IpCheckInterval))
				current := time.Now().Unix()
				if len(IPTime) > 0 {
					lock.Lock()
					for ip, second := range IPTime {
						if current-second > Cf.Msg.IpCacheTime {
							delete(IPCache, ip)
							delete(IPTime, ip)
						}
					}
					lock.Unlock()
				}
			}
		}
	}()
}

//关闭
func (I *IpService) Stop() {
	if I.DB != nil {
		L.Debug("关闭IP文件", LEVEL_INFO)
		err := I.DB.Close()
		if err != nil {
			L.Debug("关闭文件错误"+err.Error(), LEVEL_ERROR)
		}
	}
}