#!/bin/bash -
#===============================================================================
#
#          FILE: deploy.sh
#
#         USAGE: ./deploy.sh
#
#   DESCRIPTION: 
#
#       OPTIONS: ---
#  REQUIREMENTS: ---
#          BUGS: ---
#         NOTES: ---
#        AUTHOR: Lihao Zheng (https://github.com/bearzlh), zhenglh@dianzhong.com
#  ORGANIZATION: dz
#       CREATED: 2019/03/23 14时17分47秒
#      REVISION: 0.1
#===============================================================================

set -o nounset                                  # Treat unset variables as an error
if [ $# -lt 1 ] ; then
    echo "./deploy.sh version"
    exit
fi

version=$1
tar_file=package_v_$version.tar.gz
url=http://dev104.qcread.cn/$tar_file
deploy_dir=/usr/local/postlog
backup=/usr/local/postlog_`date "+%Y%m%d%H%M%S"`

cd ~

if [ -f "$tar_file" ] ; then
    rm $tar_file
fi

wget $url
tar zxf $tar_file

if [ -d $backup ] ; then
    rm -r $backup
fi

if [ -d $deploy_dir ] ; then
    cp -r $deploy_dir $backup
else
    mkdir -p $deploy_dir
fi

systemctl -l|grep postlog > /dev/null

if [ "$?" -ne "0" ] ; then
    cp package/postlog.service /usr/lib/systemd/system/
    systemctl enable postlog.service > /dev/null
fi

cp -r package/* $deploy_dir/

if [ "`hostname`" = "171-ncps" ]; then
    content=`cat $deploy_dir/config.json`
    echo $content | jq '.appid="xg"' > $deploy_dir/config.json
fi

chmod +x $deploy_dir/*.sh

rm -rf package
systemctl restart postlog
