{
  "read_path": [
    {
      "dir": "php_log_dir",//读取的日志目录
      "time_format": "Ym/d_H",//日志文件的格式
      "suffix": ".log",//日志后缀
      "type": "php",//es索引参数
      "on": true,//开启日志读取
      "continue": true,//从上次读取结束时继续读取
      "pick": "cookie,get,order,user,wechat,params,header,runtime" //采集cookie与get参数
    }
  ],
  "log": {
    "path": "/var/log/postlog/",//程序自身的日志目录
    "level": "info",//程序自身的日志目录
    "format_time": "Ym/d_H",
    "format_level": false
  },
  "factory": {
    "worker_max": 10,//线程最大数量
    "worker_init": 2, //线程初始数量
    "on": true //开启日志处理
  },
  "msg": {
    "is_batch": true,//批量发送
    "batch_size": 100,//批量发送限额
    "batch_time_second": 3,//发送时间窗口
    "send_type": "es",
    "ip_cache_time": 600,
    "ip_check_interval": 30
  },
  "monitor": {
    "cpu": 200,//cpu限制
    "load": 7,//load限制
    "sleep_interval_ns": 500,//cpu限制参数
    "sleep_time_set": 50000,//cpu限制参数
    "memory_restart": 15,//内存使用率超过此值将重启脚本
    "pick_interval": 10,//统计信息收集到es中的间隔
    "check_interval": 5 //统计信息检查间隔
  },
  "server_port": ":8081",
  "php_time_window": 2,
  "es": {
    "host": "es_host_port",//es连接地址
    "index_format": "Y.m.d", //索引时间格式
    "retry": 10, //es重试
    "concurrent_post": 10,//并发发送es请求的次数
    "Storage": "storage", //es暂存目录
    "recover_thread": 2 //es恢复暂存数据线程数
  },
  "version": "0.27",//脚本版本
  "env": "log", //用作es索引前缀，如果非空将使用-与时间分隔
  "appid": "", //appid字段，区分平台，默认使用主机名字符串
  "recover": {
    "from": "",
    "to": ""
  }
}