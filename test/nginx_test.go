package test

import (
	"fmt"
	"testing"
	"workerChannel/helper"
)

func TestRegex(t *testing.T)  {
	msg := `127.0.0.1 - - [27/Feb/2019:17:22:31 +0800] "GET /assets/js/jquery.drop.min.js?v=1551259350 HTTP/1.1" 200 3537 "http://dev.cps-pay-check.com/admin/member?addtabs=1" "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_4) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/71.0.3578.98 Safari/537.36"`
	msgMatch := helper.RegexpMatch(msg, `(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}) - - \[(\d{1,2})/(\w+)/(\d{4}):(\d{2}):(\d{2}):(\d{2}) \+0800\] "(\w+) (.*?) HTTP/1.1" (\d{3}) \d+ "(.*?)" "(.*?)"`)

	dateMap := map[string]string{"Jan":"01","Feb":"02","Mar":"03","Apr":"04","May":"05","Jun":"06","Jul":"07","Aug":"08","Sep":"09","Oct":"10","Nov":"11","Dec":"12"}
	if len(msgMatch) > 0 {
		t.Log(string(msgMatch[1]))
		timeFormat := string(msgMatch[4]) + "-" + dateMap[string(msgMatch[3])] + "-" + string(msgMatch[2]) + " " + string(msgMatch[5]) + ":" + string(msgMatch[6]) + ":" + string(msgMatch[7])
		t.Log(string(msgMatch[7]));
		t.Log(string(msgMatch[8]));
		t.Log(string(msgMatch[9]));
		t.Log(string(msgMatch[10]));
		t.Log(string(msgMatch[11]));
		t.Log(fmt.Sprintf("%d", helper.FormatTimeStamp(timeFormat, "")))
	}

}