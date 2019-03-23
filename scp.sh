#!/usr/bin/expect
set timeout 5
set package [lindex $argv 0]
spawn scp -r $package root@192.168.0.104:/var/www/cps/public
expect "*password*"
send "dianzhong\r"
expect eof
