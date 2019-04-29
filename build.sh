#!/bin/bash
os=mac
out=postlog
env=dev
dir=package
es_host_port=192.168.0.109:9200
php_log_dir=/var/log/cpslog/
deploy_dir=/usr/local/
version=`cat config.tpl | sed "s#//.*##g" | jq ".version" | sed 's/"//g'`

rm -rf $dir
mkdir -p $dir

if [ ! -z "$1" ] ; then
    out=$1
fi

if [ ! -z "$2" ] ; then
    os=$2
fi

if [ ! -z "$3" ] ; then
    env=$3
fi

if [ $env == "pro" ] ; then
    es_host_port="10.250.3.1:9200,10.250.3.2:9200,10.250.3.3:9200,10.250.3.4:9200,10.250.3.5:9200"
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

upx $dir/$out > /dev/null

cat ./config.tpl | sed "s#//.*##g" | sed "s#php_log_dir#$php_log_dir#"|sed "s#es_host_port#$es_host_port#"|jq '{read_path:[.read_path[0]],log,factory,msg,server_port,php_time_window,es,monitor,version,env}' > $dir/config.json

cp host_info.sh $dir
cp postlog.service $dir
cp deploy.sh $dir
cp controller.sh $dir
if [ "$env" != "pro" ]; then
    cat hosts.txt|grep '^dev' > ${dir}/hosts.txt
else
    cat hosts.txt|grep -v '^dev' > ${dir}/hosts.txt
fi


package_tar=package_v_${env}${version}.tar.gz
tar zcf $package_tar package
./scp.sh $package_tar > /dev/null
#rm $package_tar
rm -rf package
echo "success"