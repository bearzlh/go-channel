package service

import (
	"fmt"
	"github.com/oschwald/geoip2-golang"
	"net"
	"workerChannel/helper"
)

var DB *geoip2.Reader

func GetLocation(ip string) (string, string, string, string, string) {
	var zhou, guo, sheng, shi, jingwei = "", "", "", "", ""
	if DB == nil {
		DB, err := geoip2.Open(helper.GetPathJoin(Cf.AppPath, "./GeoLite2-City.mmdb"))
		if err != nil {
			L.Debug("error to open mmdb", LEVEL_ERROR)
			return zhou, guo, sheng, shi, jingwei
		}
		// If you are using strings that may be invalid, check that ip is not nil
		IP := net.ParseIP(ip)
		record, err := DB.City(IP)
		if err != nil {
			L.Debug("error to parse ip"+ip, LEVEL_ERROR)
			return zhou, guo, sheng, shi, jingwei
		}
		zhou = record.Continent.Names["zh-CN"]
		guo = record.Country.Names["zh-CN"]
		if len(record.Subdivisions) > 0 {
			sheng = record.Subdivisions[0].Names["zh-CN"]
		}
		shi = record.City.Names["zh-CN"]
		if record.Location.Latitude != 0 && record.Location.Longitude != 0 {
			jingwei = fmt.Sprintf("%.4f,%.4f", record.Location.Latitude, record.Location.Longitude)
		}
	}

	return zhou, guo, sheng, shi, jingwei
}