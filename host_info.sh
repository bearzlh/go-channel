#!/bin/bash -
#===============================================================================
#
#          FILE: cpu.sh
#
#         USAGE: ./cpu.sh
#
#   DESCRIPTION: 
#
#       OPTIONS: ---
#  REQUIREMENTS: ---
#          BUGS: ---
#         NOTES: ---
#        AUTHOR: Lihao Zheng (https://github.com/bearzlh), zhenglh@dianzhong.com
#  ORGANIZATION: dz
#       CREATED: 2019/03/20 15时51分10秒
#      REVISION:  ---
#===============================================================================

set -o nounset                                  # Treat unset variables as an error

keywords=postlog

getOs ()
{
    if [ ! -z "`uname -a | grep "^Darwin"`" ] ; then
        echo Mac
    else
        echo Linux
    fi
}	# ----------  end of function getOs  ----------

os=`getOs`


#---  FUNCTION  ----------------------------------------------------------------
#          NAME:  getCpu
#   DESCRIPTION:  获取当前进程cpu使用率
#    PARAMETERS:  
#       RETURNS:  
#-------------------------------------------------------------------------------
getCpu ()
{
    if [ $os == "Mac" ] ; then
        echo `top -c a -l 5|grep $keywords|awk '{print $3}'|tail -n 1`
    else
        echo `top -b -n 1|grep $keywords|awk '{print $9}'|sort -nr|head -n 1`
    fi   
}	# ----------  end of function getCpu  ----------



#---  FUNCTION  ----------------------------------------------------------------
#          NAME:  getMem
#   DESCRIPTION:  获取当前进程内存
#    PARAMETERS:
#       RETURNS:
#-------------------------------------------------------------------------------
getMem ()
{
    if [ $os != "Mac" ] ; then
        echo `top -b -n 1|grep $keywords|awk '{print $10}'|sort -nr|head -n 1`
    fi
}	# ----------  end of function getMem  ----------

#---  FUNCTION  ----------------------------------------------------------------
#          NAME:  getLoad
#   DESCRIPTION:  获取系统负载
#    PARAMETERS:  
#       RETURNS:  
#-------------------------------------------------------------------------------
getLoad ()
{
    if [ $os == "Mac" ] ; then
        echo `top -l 1|sed -n '3p'|cut -d ':' -f2|cut -d ',' -f1|sed 's/ //'`
    else
        echo `top -b -n 1|head -n 1|awk 'BEGIN{FS=": "}{print $2}'|awk 'BEGIN{FS=", "}{print $1}'`
    fi   

}	# ----------  end of function getMemory  ----------



case $1 in
    "load")
        echo `getLoad`
        ;;

    "cpu")
        echo `getCpu`
        ;;

    "memory")
        echo `getMem`
        ;;

    *)
        ;;

    esac    # --- end of case ---
