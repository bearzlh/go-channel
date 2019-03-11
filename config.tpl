{
  "read_path": [
    {
      "dir": "php_log_dir",
      "time_format": "Ym/d_H",
      "suffix": ".log",
      "type": "php",
      "on": true,
      "continue": true
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
    "path": "/var/log/postlog/",
    "level": "debug",
    "format_type": "time",
    "format": "Ym/d_H"
  },
  "factory": {
    "worker_max": 10000,
    "worker_init": 2
  },
  "msg": {
    "is_batch": true,
    "batch_size": 90,
    "batch_time_second": 3,
    "send_type": "es"
  },
  "server_port": "8081",
  "php_time_window": 2,
  "es": {
    "host": "es_host_port"
  }
}