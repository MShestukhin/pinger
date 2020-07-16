#!/usr/bin/env bash
# $1 = ["12.0.0.1", "8.8.8.8"]
# ${ip[0]} = "12.0.0.1"
# ${ip[1]} = "8.8.8.8"
IFS=, read -a ip <<< $1
if "${ip[0]}" = true ; then
	sed -i '12s/172.18.0.1/172.18.0.2/' /etc/hosts
	echo "Run server 172.18.0.2"
fi

if "${ip[1]}" = true ; then
	sed -i '12s/172.18.0.2/172.18.0.1/' /etc/hosts
	echo "Run server 172.18.0.1"
fi