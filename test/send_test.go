package test

import (
	"fmt"
	"github.com/gofrs/uuid"
	"strings"
	"testing"
	"time"
	"workerChannel/helper"
	"workerChannel/service"
)

func TestSend(t *testing.T) {
	msg := `---------------------------------------------------------------
5c88bcfe25c75 [time_time_time] 127.0.0.1 POST http://www.dev.kpread.com/admin/orders/searchorder
5c88bcfe25c75 [ info ]  [运行时间：1.591380s][吞吐率：0.63req/s] [内存消耗：9,697.58kb] [文件加载：111]
5c88bcfe25c75 [ info ] [ BEHAVIOR ] Run Closure @app_init [ RunTime:0.000358s ]
5c88bcfe25c75 [ info ] [ CACHE ] INIT File
5c88bcfe25c75 [ info ] [ BEHAVIOR ] Run Closure @app_init [ RunTime:0.004326s ]
5c88bcfe25c75 [ info ] [ LANG ] /data/www/cps/thinkphp/lang/zh-cn.php
5c88bcfe25c75 [ info ] [ ROUTE ] array (
  'type' => 'module',
  'module' => 
  array (
    0 => 'admin',
    1 => 'orders',
    2 => 'searchorder',
  ),
)
5c88bcfe25c75 [ info ] [ HEADER ] array (
  'cookie' => 'thinkphp_show_page_trace=0|0; thinkphp_show_page_trace=0|0; XDEBUG_SESSION=IDEKEY; thinkphp_show_page_trace=0|0; PHPSESSID=i3gbe4bdas6p16bte8np6ojs3e; keeplogin=1%7C604800%7C1553051329%7C42e3656dd1bb796ded598c05c480c625',
  'accept-language' => 'zh-CN,zh;q=0.9,en;q=0.8',
  'accept-encoding' => 'gzip, deflate',
  'referer' => 'http://www.dev.kpread.com/admin/orders/searchorder?addtabs=1',
  'accept' => 'text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8',
  'user-agent' => 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.121 Safari/537.36',
  'content-type' => 'application/x-www-form-urlencoded',
  'upgrade-insecure-requests' => '1',
  'origin' => 'http://www.dev.kpread.com',
  'cache-control' => 'max-age=0',
  'content-length' => '47',
  'connection' => 'keep-alive',
  'host' => 'www.dev.kpread.com',
)
5c88bcfe25c75 [ info ] [ PARAM ] array (
  'oid' => '4200000231201810315752798752',
  'lastname' => 'Mouse',
)
5c88bcfe25c75 [ info ] [ LANG ] /data/www/cps/public/../application/admin/lang/zh-cn.php
5c88bcfe25c75 [ info ] [ DB ] INIT mysql
5c88bcfe25c75 [ info ] Redis Connect Info: Ip:192.168.0.105 Port:16381 No:1
5c88bcfe25c75 [ info ] [ BEHAVIOR ] Run app\common\behavior\Common @module_init [ RunTime:0.418286s ]
5c88bcfe25c75 [ info ] [ SESSION ] INIT array (
  'auto_start' => true,
  'domain' => NULL,
)
5c88bcfe25c75 [ info ] OS:Other UA:Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.121 Safari/537.36 NetType:Other IP:127.0.0.1 [0|0|0|内网IP|内网IP] admin_id:1 group:1 REMOTE_ADDR:127.0.0.1 HTTP_X_FORWARDED_FOR:
5c88bcfe25c75 [ info ] SSDB连接,host:192.168.0.154,port:18888,pwd:273813e5f4041f6dce947bd06b737dac
5c88bcfe25c75 [ info ] SSDB连接,host:192.168.0.154,port:28888,pwd:273813e5f4041f6dce947bd06b737dac
5c88bcfe25c75 [ info ] [ RUN ] app\admin\controller\Orders->searchorder[ /data/www/cps/application/admin/controller/Orders.php ]
5c88bcfe25c75 [ info ] [ VIEW ] /data/www/cps/public/../application/admin/view/orders/searchorder.html [ array (
  0 => 'backend_group',
  1 => 'breadcrumb',
  2 => 'site',
  3 => 'config',
  4 => 'auth',
  5 => 'admin',
  6 => 'typeList',
  7 => 'stateList',
  8 => 'res',
) ]
5c88bcfe25c75 [ info ] [ BEHAVIOR ] Run app\admin\behavior\AdminLog @app_end [ RunTime:0.020256s ]
5c88bcfe25c75 [ info ] [ LOG ] INIT app\driver\log\File
5c88bcfe25c75 [ sql ] [ DB ] CONNECT:[ UseTime:0.135579s ] mysql:host=192.168.0.104;port=3306,3306;dbname=test_cps;charset=utf8mb4
5c88bcfe25c75 [ sql ] [ SQL ] SHOW COLUMNS FROM auth_group_access [ RunTime:0.048940s ]
5c88bcfe25c75 [ sql ] [ SQL ] SELECT aga.uid,aga.group_id,ag.id,ag.pid,ag.name,ag.rules FROM auth_group_access aga LEFT JOIN auth_group ag ON aga.group_id=ag.id WHERE  (  aga.uid='1' and ag.status='normal' ) [ RunTime:0.187050s ]
5c88bcfe25c75 [ sql ] [ SQL ] SHOW COLUMNS FROM auth_rule [ RunTime:0.059531s ]
5c88bcfe25c75 [ sql ] [ SQL ] SELECT id,pid,condition,icon,name,title,ismenu FROM auth_rule WHERE  status = 'normal'  AND id IN (1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,41,42,43,44,45,46,47,48,49,50,51,52,53,54,55,56,57,58,59,60,61,62,63,64,65,122,123,125,127,129,130,132,133,134,231,233,234,235,236,237,238,239,240,242,243,244,245,246,247,248,249,253,256,257,261,262,263,264,265,266,267,268,269,270,304,347,348,349,350,371,372,373,374,376,377,378,379,380,381,382,383,384,385,386,387,388,389,390,391,392,394,399,401,403,404,405,406,407,410,419,435,436,437,438,439,441,450,451,452,453,454,455,456,457,458,459,460,461,494,495,496,497,498,511,512,513,514,515,516,524,525,526,527,528,529,530,531,532,533,534,535,536,537,538,539,540,541,542,543,544,545,546,547,548,549,550,551,552,553,554,555,556,557,558,559,560,561,562,563,564,567,568,577,578,579,580,646,698,700,701,702,703,704,705,706,707,709,710,711,712,713,714,715,716,717,718,719,720,733,735,736,743,785,788,789,790,791,799,800,802,803,804,812,814,815,822,824,825,839,845,846,847,848,849,850,851,852,853,854,855,856,857,858,859,860,861,862,863,865,866,867,868,878,879,880,881,882,883,884,885,886,887,888,889,890,891,892,893,120,121,126,229,241,152,232,230,696,697,699,737,672) [ RunTime:0.025310s ]
5c88bcfe25c75 [ sql ] [ SQL ] SHOW COLUMNS FROM admin_extend [ RunTime:0.033663s ]
5c88bcfe25c75 [ sql ] [ SQL ] SELECT * FROM admin_extend WHERE  admin_id = 1 LIMIT 1 [ RunTime:0.130323s ]
5c88bcfe25c75 [ sql ] [ SQL ] SELECT * FROM auth_group_access WHERE  uid = 1 LIMIT 1 [ RunTime:0.005637s ]
5c88bcfe25c75 [ sql ] [ SQL ] SHOW COLUMNS FROM orders [ RunTime:0.007260s ]
5c88bcfe25c75 [ sql ] [ SQL ] SELECT * FROM orders WHERE  transaction_id IN ('4200000231201810315752798752') [ RunTime:0.098433s ]
5c88bcfe25c75 [ sql ] [ SQL ] SHOW COLUMNS FROM wxpay [ RunTime:0.100760s ]
5c88bcfe25c75 [ sql ] [ SQL ] SELECT id,mcid,quartet_app_id FROM wxpay [ RunTime:0.084766s ]
5c88bcfe25c75 [ sql ] [ SQL ] SHOW COLUMNS FROM admin_log [ RunTime:0.008542s ]
5c88bcfe25c75 [ sql ] [ SQL ] INSERT INTO admin_log (title , content , url , admin_id , username , useragent , ip , createtime) VALUES ('四方投诉订单查询' , '{\"oid\":\"4200000231201810315752798752\",\"lastname\":\"Mouse\"}' , '/admin/orders/searchorder' , 1 , 'admin' , 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.121 Safari/537.36' , '127.0.0.1' , 1552465149) [ RunTime:0.009649s ]
5c88bcfe25c75 [ debug ] time:[ 2019-03-13 16:19:09 ]	pid:[ 2324 ]	{"code":0,"msg":"","data":"-","debug":[{"file":"\/data\/www\/cps\/application\/main\/service\/AdminService.php","line":276,"function":"getReturn","class":"app\\main\\service\\BaseService","args":[]}]}
`
	hourDoc := int64(200000)
	DocMax := hourDoc / 30
	count := int64(0)
	sleep := time.Hour / time.Duration(hourDoc)
	timeUsed := time.Duration(0)
	for {
		count++
		if count > DocMax {
			break
		}
		select {
		case <-time.After(sleep - timeUsed):
			start := time.Now().UnixNano()
			dateLog := helper.TimeFormat("d_H", 0)
			timeNow := time.Now().Format("2006-01-02 15:04:05")
			id, _ := uuid.NewV4()
			sub := strings.Replace(id.String(), "-", "", 10)
			tmp := strings.Replace(msg, "time_time_time", timeNow, 10)
			low := len(sub) - 13
			service.L.Debug(fmt.Sprintf("%s %s %d", sub[low:], dateLog, sleep - timeUsed), service.LEVEL_DEBUG)
			tmp = strings.Replace(tmp, "5c88bcfe25c75", sub[low:], 37)
			helper.FilePutContents("/data/www/cps/runtime/log/201903/"+dateLog+".log", tmp, true)
			end := time.Now().UnixNano()
			timeUsed = time.Duration(end - start)
		}
	}
}