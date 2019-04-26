#!/bin/bash -
#===============================================================================
#
#          FILE: controller1.sh
#
#         USAGE: ./controller1.sh
#
#   DESCRIPTION:
#
#       OPTIONS: ---
#  REQUIREMENTS: ---
#          BUGS: ---
#         NOTES: ---
#        AUTHOR: Lihao Zheng (https://github.com/bearzlh), zhenglh@dianzhong.com
#  ORGANIZATION: dz
#       CREATED: 2019/04/23 11时53分53秒
#      REVISION:  0.1
#===============================================================================

set -o nounset                                  # Treat unset variables as an error

cd /usr/local/postlog

#---   ----------------------------------------------------------------
#          NAME:  getOs
#   DESCRIPTION:  获取系统版本
#    PARAMETERS:  
#       RETURNS:  
#-------------------------------------------------------------------------------
getOs ()
{
    if [ ! -z "`uname -a | grep "^Darwin"`" ] ; then
        echo Mac
    else
        echo Linux
    fi
}	# ----------  end of getOs  ----------

os=`getOs`

jq --version>/dev/null 2>&1

if [[ "$?" != "0" ]]; then
    echo "jq depend, installing"
    if [[ "$os" = "Mac" ]]; then
        brew install jq
    else
        wget http://dev104.qcread.cn/jq && chmod +x jq && mv ./jq /usr/bin/
    fi
fi

#---   ----------------------------------------------------------------
#          NAME:  host_action
#   DESCRIPTION:  遍历主机，调用接口
#    PARAMETERS:  
#       RETURNS:  
#-------------------------------------------------------------------------------
host_action() {
    action=$1
    args=$2
    filter=$3
    for host in `cat ./hosts.txt|grep -e $filter` ; do
        nickname=`echo $host | cut -d '#' -f1`
        hostname=`echo $host | cut -d '#' -f2`
        $action $nickname $hostname $args
    done
}

#---   ----------------------------------------------------------------
#          NAME:  info
#   DESCRIPTION:  获取状态页信息
#    PARAMETERS:  
#       RETURNS:  
#-------------------------------------------------------------------------------
info() {
    nickname=$1
    hostname=$2
    field=$3
    echo -e $nickname `curl -s "http://$hostname:8081/status" | jq .${field}`
}

#---   ----------------------------------------------------------------
#          NAME:  update
#   DESCRIPTION:  更新版本
#    PARAMETERS:  
#       RETURNS:  
#-------------------------------------------------------------------------------
update() {
    nickname=$1
    hostname=$2
    version=$3
    echo "updating $nickname..."
    curl -s "http://$hostname:8081/update?version=${version}"
    for i in `seq 1 5` ; do
        sleep 1
        curl -s "http://$hostname:8081">/dev/null 2>&1
        if [[ "$?" == "0" ]]; then
            echo "updated successfully"
            break
        fi
    done

}


#---  FUNCTION  ----------------------------------------------------------------
#          NAME:  stop
#   DESCRIPTION:  停止日志采集
#    PARAMETERS:  
#       RETURNS:  
#-------------------------------------------------------------------------------
stop ()
{
    nickname=$1
    hostname=$2
    echo "stopping $nickname..."
    curl -s "http://$hostname:8081/stop"
}	# ----------  end of stop  ----------


#---  FUNCTION  ----------------------------------------------------------------
#          NAME:  restart
#   DESCRIPTION:  重启服务
#    PARAMETERS:  
#       RETURNS:  
#-------------------------------------------------------------------------------
restart ()
{
    nickname=$1
    hostname=$2
    echo "restarting $nickname..."
    curl -s "http://$hostname:8081/restart"

}	# ----------  end of function restart  ----------


config ()
{
    nickname=$1
    hostname=$2
    key=$3
    value=$4
    echo "update $nickname..."
    curl -s "http://$hostname:8081/config?key=${key}&value=${value}"
}	# ----------  end of function config  ----------


#---  FUNCTION  ----------------------------------------------------------------
#          NAME:  update_config
#   DESCRIPTION:  配置文件更新
#    PARAMETERS:  
#       RETURNS:  
#-------------------------------------------------------------------------------
update_config ()
{
    key=$1
    value=$2
    content=`cat ./config.json`
    echo $content | jq '.'$key'="'$value'"'
    if [ "$?" = "0" ] ; then
        if [ "`echo $value | grep '^[0-9]'`" != "" ]; then
            echo $content | jq '.'$key'='$value'' > ./config.json
        else
            echo $content | jq '.'$key'="'$value'"' > ./config.json
        fi
        echo "success"
    fi
}	# ----------  end of function update_config  ----------

case $1 in
    "info")
        if [ "$#" = "3" ]; then
            filter=$3
        else
            filter=".*"
        fi
        host_action info $2 ${filter}
        ;;
    "stop")
        if [ "$#" = "2" ]; then
            host_action stop "" $2
        else
            systemctl stop postlog
        fi
        ;;
    "restart")
        if [ "$#" = "2" ]; then
            host_action restart "" $2
        else
            systemctl restart postlog
        fi
        ;;
    "config")
        if [ "$#" = "4" ]; then
            filter=$4
        else
            filter=".*"
        fi
        key=$2
        value=$3
        echo "config before"
        host_action info cf.$key ${filter}
        host_action config "$2 $3" ${filter}
        sleep 1
        echo "update after"
        host_action info cf.$key ${filter}
        ;;
    "update_config")
        update_config $2 $3
        ;;
    "update")
        if [ "$#" = "3" ]; then
            filter=$3
        else
            filter=".*"
        fi
        echo "update before"
        host_action info cf.version ${filter}
        host_action update $2 ${filter}
        echo "update after"
        host_action info cf.version ${filter}
        ;;
    *)
     echo "./controller.sh info|update args"
        ;;
    esac    # --- end of case ---
