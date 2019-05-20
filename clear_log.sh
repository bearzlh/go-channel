#!/bin/bash -
#===============================================================================
#
#          FILE: clear_log.sh
#
#         USAGE: ./clear_log.sh
#
#   DESCRIPTION: 
#
#       OPTIONS: ---
#  REQUIREMENTS: ---
#          BUGS: ---
#         NOTES: ---
#        AUTHOR: Lihao Zheng (https://github.com/bearzlh), zhenglh@dianzhong.com
#  ORGANIZATION: dz
#       CREATED: 2019/05/20 10时57分00秒
#      REVISION:  ---
#===============================================================================
set -o nounset                                  # Treat unset variables as an error


action=help
config_file=/usr/local/postlog/config.json
if [ $# -ge 1 ] ; then
    action=$1
fi

if [ $# -ge 2 ] ; then
    config_file=$2
fi

keep_days ()
{
    day_to_keep=`cat $config_file|jq .log.log_keep_day|sed 's/"//g'`
    log_dir=`cat $config_file|jq .log.path|sed 's/"//g'|sed 's#/$##g'`
    first_day_to_keep=`date -d "$day_to_keep days ago" "+%Y%m%d%H"`

    for dir in `ls $log_dir`; do
        if [ ! -d $log_dir/$dir ] ; then
            rm $log_dir/$dir
        fi
        for file in `ls $log_dir/$dir`; do
            file_path=$log_dir/$dir/$file
            file_time=`echo $dir$file|sed 's/[^0-9]//g'`
            if [[ $file_time -le $first_day_to_keep ]] ; then
                rm $file_path
            fi
        done

        if [ -d $log_dir/$dir -a "`ls $log_dir/$dir|wc -l`" == "" ] ; then
            rm -rf $log_dir/$dir
        fi
    done
}	# ----------  end of function keep_days  ----------

keep_disk ()
{
    disk_to_keep=`cat $config_file|jq .log.log_keep_m|sed 's/"//g'`
    log_dir=`cat $config_file|jq .log.path|sed 's/"//g'|sed 's#/$##g'`
    file_keep_m=0

    for dir in `ls $log_dir|sort -r`; do
        if [ ! -d $log_dir/$dir ] ; then
            rm $log_dir/$dir
        fi
        for file in `ls $log_dir/$dir|sort -r`; do
            file_path=$log_dir/$dir/$file
            file_disk=`du -s $file_path|awk '{print $1}'`
            file_disk_m=$[$file_disk/1024]
            if [[ $[$file_disk_m+$file_keep_m] -gt $disk_to_keep ]] ; then
                rm $file_path
            else
                file_keep_m=$[$file_disk_m+$file_keep_m]
            fi
        done

        if [ -d $log_dir/$dir -a "`ls $log_dir/$dir|wc -l`" == "" ] ; then
            rm -rf $log_dir/$dir
        fi
    done
}	# ----------  end of function keep_disk  ----------

case $action in
    "disk")
        keep_disk
        ;;

    "days")
        keep_days
        ;;

    "help")
        echo "$0 [disk|days]"
        ;;

    esac    # --- end of case ---
