#!/usr/bin/env bash
# ["10.10.10.6", "10.10.10.70"]
# $1 = [5,5]
# ${ip[0]} = "10.10.10.6" = 5
# ${ip[1]} = "10.10.10.70" = 5
IFS=, read -a ip <<< $1

if [[ "${ip[0]}" = 5 && "${ip[1]}" = 5 ]] || [[ "${ip[0]}" != 5 && "${ip[1]}" != 5 ]]; then
        echo "Nothing change ..."
  else
    if [[ "${ip[0]}" = 5 ]] ; then
        sudo ./ip_change /etc/hosts active-dbvip 10.10.10.70 10.10.10.6
        sudo ./ip_change /etc/hosts active-webvip 10.10.10.73 10.10.10.9
        sudo ./ip_change /etc/hosts active-db1 10.10.10.68 10.10.10.4
        sudo ./ip_change /etc/hosts active-db1 10.10.10.69 10.10.10.5
        sudo ./ip_change /etc/hosts active-web1 10.10.10.71 10.10.10.7
        sudo ./ip_change /etc/hosts active-web1 10.10.10.72 10.10.10.8
        echo "Change /etc/host : Run server 10.10.10.6"
    fi

    if [[ "${ip[1]}" = 5 ]] ; then
        sudo ./ip_change /etc/hosts active-dbvip 10.10.10.6 10.10.10.70
        sudo ./ip_change /etc/hosts active-webvip 10.10.10.9 10.10.10.73
        sudo ./ip_change /etc/hosts active-db1 10.10.10.4 10.10.10.68
        sudo ./ip_change /etc/hosts active-db1 10.10.10.5 10.10.10.69
        sudo ./ip_change /etc/hosts active-web1 10.10.10.7 10.10.10.71
        sudo ./ip_change /etc/hosts active-web1 10.10.10.8 10.10.10.72
        echo "Change /etc/host : Run server 10.10.10.70"
    fi
fi
