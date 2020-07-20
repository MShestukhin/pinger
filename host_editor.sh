#!/usr/bin/env bash
# ["10.10.10.6", "10.10.10.70"]
# $1 = [5,5]
# ${ip[0]} = "10.10.10.6" = 5
# ${ip[1]} = "10.10.10.70" = 5
IFS=, read -a ip <<< $1

if [[ "${ip[0]}" = 5 && "${ip[1]}" = 5 ]]; then
  sudo -u svyazcom cp -f etc/hostnet1acctive /etc/host
  echo "Change /etc/host : Run server 10.10.10.6"
  else
    if [[ "${ip[0]}" = 5 ]] ; then
      sudo -u svyazcom cp -f etc/hostnet1acctive /etc/host
      echo "Change /etc/host : Run server 10.10.10.6"
    fi

    if [[ "${ip[1]}" = 5 ]] ; then
      sudo -u svyazcom cp -f etc/hostnet2acctive /etc/host
      echo "Change /etc/host : Run server 10.10.10.70"
    fi
fi

if [[ "${ip[0]}" = 5 && "${ip[1]}" = 5 ]]; then
  ln -sfT /opt/svyazcom/etc/replica_backup.conf /opt/svyazcom/etc/replica.conf;
  sudo -u svyazcom /opt/svyazcom/sbin/replica_loop.sh restart
	echo "Replica: All servers run!"
  else
  ln -sfT /opt/svyazcom/etc/replica_fallback.conf /opt/svyazcom/etc/replica.conf;
  sudo -u svyazcom /opt/svyazcom/sbin/replica_loop.sh restart
	echo "Replica: Not all servers run!"
fi