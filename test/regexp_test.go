package test

import (
	"encoding/json"
	"github.com/bitly/go-simplejson"
	"regexp"
	"strconv"
	"strings"
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
	} else {
		t.Log(0)
	}
}

func TestLevel(t *testing.T) {
	wechatMatch := helper.RegexpMatch("wxasfds.aa.com", `^(wx\w+)\.`)
	t.Log(string(wechatMatch[0]), string(wechatMatch[1]))
}

func TestAppId(t *testing.T) {
	for {
		service.L.Debug(time.Now().String(), "debug")
	}
}

func TestPhp(t *testing.T) {
	msg := `[1] 5c88bcfe25c75 [ info ] [ BEHAVIOR ] Run app\admin\behavior\AdminLog @app_end [ RunTime:0.020256s ]`
	mlist := helper.RegexpMatch(msg, service.PhpMsgRegex)
	for _, item := range mlist {
		t.Log(string(item))
	}
}

func TestCookie(t *testing.T) {
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

func TestCookie1(t *testing.T) {
	msg := `OS:Other UA:Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.100 Safari/537.36 NetType:Other IP:220.201.216.47 [中国|0|黑龙江省|哈尔滨市|联通|854] admin_id:14749 group:3 REMOTE_ADDR:10.250.3.48 HTTP_X_FORWARDED_FOR:220.201.216.47 [ FromPreLog:0.057767 ]`
	//regex := `OS:.*?[(\w+)|\w+|(\w+)|(\w+)|(\w+)]user_id:(\d+)* .*? channel_id:(\d+)* agent_id:(\d+)* referral_id:(\d+)*`
	regex := service.PhpAdminCookie
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

func TestDatabase(t *testing.T) {
	msg := `get_db_connect table:user params:664785048`
	if strings.Contains(service.Cf.ReadPath[0].Pick, "user") {
		if len(msg) > 13 && msg[0:14] == "get_db_connect" {
			res := helper.RegexpMatch(msg, `get_db_connect table:(\w+) params:(\d+)`)
			if len(res) > 0 {
				tableName := string(res[1])
				if strings.Contains(service.UserTable, tableName) {
					t.Log(string(res[2]))
				}
			}
		}
	}

}

func TestOrder(t *testing.T) {
	//msg := "wechatpay_create_order_success!wxpay_id:5,wxpay_name:奇异书阁,mch_id:1518063301,channel_id:7234,user_id:5831708,money:50.00,good_id:39,out_trade_no:20190429133827_5831708_4zS2,api_run_time:0.223 s"
	msgcallback := "wechatpay_callback_success!wxpay_id:1,channel_id:1778,money:0.01,good_id:3,out_trade_no:20190429155943_10131_iWfa"
	//regex := `\w+_\w+_\w)?+_\w+!wxpay_id:(.*?),wxpay_name:.*?,mch_id:.*?,channel_id:(.*?),user_id:(.*?),money:(.*?),good_id:(.*?),out_trade_no:.*?`
	regexcallback := `(\w+)_callback_(\w+)!wxpay_id:(.*?),channel_id:(.*?),money:(.*?),good_id:(.*?),.*?`
	res := helper.RegexpMatch(msgcallback, regexcallback)
	if len(res) > 0 {
		for _, item := range res {
			t.Log(string(item))
		}
	} else {
		t.Log(0)
	}
}

func TestDomainPort(t *testing.T) {
	msg := `px-cb028-wx217270d72b4bffef-8311387-46633.yifengaq.cn:443`
	regex := `^px-\w+-(\w+)-(\d+)-\w+\..*`
	res := helper.RegexpMatch(msg, regex)
	if len(res) > 0 {
		for _, item := range res {
			t.Log(string(item))
		}
	} else {
		t.Log(0)
	}
}

func TestUri(t *testing.T) {
	msg := `/api/wechat/mpapi/appid/wx2472aed3807f6ae9`
	regex := `^/api/wechat/mpapi/appid/(\w+)`
	res := helper.RegexpMatch(msg, regex)
	if len(res) > 0 {
		for _, item := range res {
			t.Log(string(item))
		}
	} else {
		t.Log(0)
	}
}

func TestFirstLine(t *testing.T) {
	line := "[ SQL ] SELECT * FROM `client_config` WHERE  `fun_type` = '1'  AND `status` = '1'  AND `start_time` < '2019-08-26 13:45:54'  AND `end_time` > '2019-08-26 13:45:54'  AND `user_pay_type` IN ('0')  AND `version` IN ('1','-1') ORDER BY `sort`  desc LIMIT 5 [ RunTime:0.001141s ]"
	res := helper.RegexpMatch(line, `^\[ (\w+) \] .*? \[ RunTime:(\d+\.\d+)s \]`)
	m := map[string]float64{"SQL": 0.0001}
	if len(res) > 0 {
		key := string(res[1])
		value, _ := strconv.ParseFloat(res[2], 64)
		if exists, ok := m[key]; ok {
			if exists < value {
				m[key] = value
			}
		} else {
			m[key] = value
		}
		t.Log(m)

	} else {
		t.Log(0)
	}
}

func TestWechatMsg(t *testing.T) {
	msg := `time:[ 2019-05-14 13:46:55 ]\tpid:[ 17256 ]\t[ WeChat ] [ MP ] [ API ] Message: `
	split_str := `[ WeChat ] [ MP ] [ API ] Message: `
	if strings.Contains(msg, split_str) {
		list := strings.Split(msg, split_str)
		wechatString := strings.Replace(list[1], `\"`, `"`, 100)
		WechatMsg := new(object.WechatMsg)
		t.Log(wechatString)
		err := json.Unmarshal([]byte(wechatString), WechatMsg)
		if err != nil {
			t.Log(err.Error())
		}
		t.Log(WechatMsg)
	}
}

func TestRunTime(t *testing.T) {
	msg := "[运行时间：0.071368s][吞吐率：14.01req/s] [内存消耗：8,440.49kb] [文件加载：105]"
	regex := `^\[运行时间：(\d+.\d+)s\]\[吞吐率：(\d+.\d+)req/s\] \[内存消耗：((\d+,)?\d+.\d+)kb\] \[文件加载：(\d+)\]`
	res := helper.RegexpMatch(msg, regex)
	if len(res) > 0 {
		t.Log(len(res))
		for k, v := range res {
			t.Log(k, v)
		}
	} else {
		t.Log(0)
	}
}

func TestHeader(t *testing.T) {
	msg := `{}`
	value, _ := simplejson.NewJson([]byte(msg))
	uid, _ := value.Get("uid").String()
	t.Log(uid)
}
