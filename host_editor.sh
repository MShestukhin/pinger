#!/usr/bin/env bash
# ["77.88.8.8", "8.8.8.8"]
# $1 = [5,5]
# ${ip[0]} = "77.88.8.8"
# ${ip[1]} = "8.8.8.8"
IFS=, read -a ip <<< $1

if [[ "${ip[0]}" = 5 && "${ip[1]}" = 5 ]]; then
  ln -sf ./hostnet1acctive /etc/host
  echo "Run server 77.88.8.8"
  else
    if [[ "${ip[0]}" = 5 ]] ; then
      ln -sf ./hostnet1acctive /etc/host
      echo "Run server 77.88.8.8"
    fi

    if [[ "${ip[1]}" = 5 ]] ; then
      ln -sf ./hostnet2acctive /etc/host
      echo "Run server 8.8.8.8"
    fi
fi