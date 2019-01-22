#!/bin/sh

set -e

# Initial installation: $1 == 1
# Upgrade: $1 == 2, and configured to restart on upgrade
#if [ $1 -eq 1 ] ; then
  if ! getent group "log-collector" > /dev/null 2>&1 ; then
    groupadd -r "log-collector"
  fi
  if ! getent passwd "log-collector" > /dev/null 2>&1 ; then
    useradd -r -g log-collector -d /usr/share/log-collector -s /sbin/nologin \
      -c "log collector user" log-collector
  fi
  chown -R log-collector /var/lib/nginx-log-collector
  chown -R log-collector /var/log/nginx-log-collector
  /bin/systemctl daemon-reload
  /bin/systemctl enable nginx-log-collector.service
#fi