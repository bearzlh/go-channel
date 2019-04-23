#!/bin/bash -
#===============================================================================
#
#          FILE: stop.sh
#
#         USAGE: ./stop.sh
#
#   DESCRIPTION: 
#
#       OPTIONS: ---
#  REQUIREMENTS: ---
#          BUGS: ---
#         NOTES: ---
#        AUTHOR: Lihao Zheng (https://github.com/bearzlh), zhenglh@dianzhong.com
#  ORGANIZATION: dz
#       CREATED: 2019/04/23 12时01分14秒
#      REVISION:  ---
#===============================================================================

set -o nounset                                  # Treat unset variables as an error

systemctl $1 postlog
