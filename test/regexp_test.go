package test

import (
	"regexp"
	"testing"
	"time"
	"workerChannel/helper"
	"workerChannel/object"
	"workerChannel/service"
)

func TestRegexp(t *testing.T) {
	bid := []byte("5c45b45c38751 [ info ] 获取缓存:PF:1 filecache命中")
	reg := regexp.MustCompile(`^[[:alnum:]]{13} `)
	s := reg.Find(bid)
	if string(s) == "" {
		t.Log(1)
	}else{
		t.Log(0)
	}
}

func TestLevel(t *testing.T) {
	wechatMatch := helper.RegexpMatch("wxasfds.aa.com", `^(wx\w+)\.`)
	t.Log(string(wechatMatch[0]), string(wechatMatch[1]))
}

func TestAppId(t *testing.T) {
	for  {
		service.L.Debug(time.Now().String(), "debug")
	}
}

func TestJson(t *testing.T) {
	nginxStr := `[100] 127.0.0.1 - - [09/Mar/2019:16:41:15 +0800] "www.dev.kpread.com:80" "GET /admin/auth/agent?ref=addtabs HTTP/1.1 status:302 cost:0.478 php:0.478 5" "http://www.dev.kpread.com/admin/templatemessage?ref=addtabs" "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.119 Safari/537.36" "-"127.0.0.1:9000`

	NginxMsg := object.NginxMsg{}
	service.ProcessNginxMsg(&NginxMsg, nginxStr)
	t.Log(NginxMsg)
}

func TestPhp(t *testing.T) {
	msg := `[1] 5c88bcfe25c75 [ info ] [ BEHAVIOR ] Run app\admin\behavior\AdminLog @app_end [ RunTime:0.020256s ]`
	mlist := helper.RegexpMatch(msg, service.PhpMsgRegex)
	for _, item := range mlist {
		t.Log(string(item))
	}
}

func TestCookie(t *testing.T)  {
	msg := `OS:Android UA:Mozilla/5.0 (Linux; Android 6.0.1; OPPO R9s Build/MMB29M; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/66.0.3359.126 MQQBrowser/6.2 TBS/044506 Mobile Safari/537.36 MMWEBID/1091 MicroMessenger/7.0.3.1400(0x2700033B) Process/tools NetType/WIFI Language/zh_CN NetType:4G IP:117.179.11.226 [中国|0|黑龙江省|哈尔滨市|移动] user_id:1 openid:oPUdp1H58SW4taFN3fzR8mu4A-d8 channel_id:deleted agent_id:deleted referral_id:deleted REMOTE_ADDR:10.250.2.8 HTTP_X_FORWARDED_FOR:117.179.11.226`
	regex := service.PhpFrontCookie
	res := helper.RegexpMatch(msg, regex)
	if len(res) > 0 {
		for _, item := range res {
			t.Log(string(item))
		}
	} else {
		t.Log(0)
	}
}

func TestCookie1(t *testing.T)  {
	msg := `OS:Other UA:Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/70.0.3538.25 Safari/537.36 Core/1.70.3638.400 QQBrowser/10.4.3264.400 NetType:Other IP:223.74.237.155 [中国|0|广东省|揭阳市|移动] admin_id:deleted group: REMOTE_ADDR:10.250.2.7 HTTP_X_FORWARDED_FOR:223.74.237.155`
	//regex := `OS:.*?[(\w+)|\w+|(\w+)|(\w+)|(\w+)]user_id:(\d+)* .*? channel_id:(\d+)* agent_id:(\d+)* referral_id:(\d+)*`
	regex := `.*?\[(.*?)\|0\|(.*?)\|(.*?)\|(.*?)\] admin_id:(\d+)* group:(\d+)* `
	res := helper.RegexpMatch(msg, regex)
	if len(res) > 0 {
		for _, item := range res {
			t.Log(string(item))
		}
	} else {
		t.Log(0)
	}

	t.Log(msg[0:3])
}