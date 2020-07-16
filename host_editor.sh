#!/usr/bin/env bash
# $1 = ["12.0.0.1", "8.8.8.8"]
# ${ip[0]} = "12.0.0.1"
# ${ip[1]} = "8.8.8.8"
IFS=, read -a ip <<< $1
if "${ip[0]}" = true ; then
  ln -sf ./hostnet1acctive /etc/host
	echo "Run server 12.0.0.1"
fi

if "${ip[1]}" = true ; then
  ln -sf ./hostnet2acctive /etc/host
	echo "Run server 8.8.8.8"
fi