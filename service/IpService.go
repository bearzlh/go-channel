package service

import (
	"io/ioutil"
	"net"
	"net/http"
	"os"
)

type IpService struct {
	InternalIp string
	ExternalIp string
}

var Ip *IpService

var chIp = make(chan bool, 1)

func GetIp() *IpService {
	chIp <- true
	if Ip == nil {
		Ip = &IpService{}
		Ip.InternalIp = Ip.GetInternalIp()
		Ip.ExternalIp = Ip.GetExternalIp()
	}
	<- chIp

	return Ip
}

func (I *IpService) GetExternalIp() string {
	resp, err := http.Get("http://myexternalip.com/raw")
	if err != nil {
		return ""
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			L.Debug(err.Error(), LEVEL_ERROR)
		}
	}()
	content, _ := ioutil.ReadAll(resp.Body)
	return string(content)
}

func (I *IpService) GetInternalIp() string {
	address, err := net.InterfaceAddrs()

	if err != nil {
		L.Debug(err.Error(), LEVEL_ERROR)
		os.Exit(1)
	}

	for _, address := range address {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return ""
}