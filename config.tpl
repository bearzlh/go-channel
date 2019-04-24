{
  "read_path": [
    {
      "dir": "php_log_dir",//读取的日志目录
      "time_format": "Ym/d_H",//日志文件的格式
      "suffix": ".log",//日志后缀
      "type": "php",//es索引参数
      "on": true,//开启日志读取
      "continue": true,//从上次读取结束时继续读取
      "pick": "cookie,get" //采集cookie与get参数
    },
    {
      "dir": "{nginx_log_dir}",
      "time_format": "",
      "suffix": "access.log",
      "type": "nginx",
      "on": false,
      "continue": true
    }
  ],
  "log": {
    "path": "/var/log/postlog/",//程序自身的日志目录
    "level": "info",//程序自身的日志目录
    "format_type": "time",
    "format": "Ym/d_H"
  },
  "factory": {
    "worker_max": 10,//线程最大数量
    "worker_init": 4 //计算机核数
  },
  "msg": {
    "is_batch": true,//批量发送
    "batch_size": 200,//批量发送限额
    "batch_time_second": 2,//发送时间窗口
    "send_type": "es"
  },
  "monitor": {
    "cpu": 150,//cpu限制
    "sleep_interval_ns": 200,//cpu限制参数
    "memory_restart": 15,//内存使用率超过此值将重启脚本
    "pick_interval": 15,//统计信息收集到es中的间隔
    "check_interval": 5 //统计信息检查间隔
  },
  "server_port": "8081",
  "php_time_window": 2,
  "es": {
    "host": "es_host_port",//es连接地址
    "index_format": "Y.m.d" //索引时间格式
  },
  "version": "0.11",//脚本版本
  "env": "log" //用作es索引前缀，如果非空将使用-与时间分隔
}