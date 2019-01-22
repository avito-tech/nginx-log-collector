#!/bin/sh

set -e

# Initial installation: $1 == 1
# Upgrade: $1 == 2, and configured to restart on upgrade
#if [ $1 -eq 1 ] ; then
  /bin/systemctl stop nginx-log-collector.service
#fi