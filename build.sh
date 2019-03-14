#!/bin/bash
os=mac
out=postlog
dir=package
es_host_port=192.168.0.109:9200
php_log_dir=/var/log/cpslog/
host_list="192.168.0.104 192.168.0.24 192.168.0.25 192.168.0.154"
deploy_dir=/usr/local/

rm -rf $dir
mkdir -p $dir

if [ ! -z "$1" ] ; then
    out=$1
fi

if [ ! -z "$2" ] ; then
    os=$2
fi

if [ $os == "mac" ] ; then
    echo "building for $os"
    go build -o $dir/$out main.go
elif [ $os == "linux" ] ;  then
    echo "building for $os"
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $dir/$out main.go
elif [ $os == "windows" ] ; then
    echo "building for $os"
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o $dir/$out main.go
else
    echo "./build [target] [mac|linux|windows]\n"
    exit
fi

upx $dir/$out

cat ./config.tpl | sed "s#php_log_dir#$php_log_dir#"|sed "s#es_host_port#$es_host_port#"|jq '{read_path:[.read_path[0]],log,factory,msg,server_port,php_time_window,es,monitor}' > $dir/config.json

cp host_info.sh $dir
echo "success in $dir"