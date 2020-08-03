#!/usr/bin/env bash
IFS=, read -a ip <<< $1
if [[ "${ip[0]}" = 5 && "${ip[1]}" = 5 ]] || [[ "${ip[2]}" = 5 && "${ip[3]}" = 5 ]]; then
  	ln -sfT ./replica_backup.conf /opt/svyazcom/etc/replica.conf
	sudo -u svyazcom /opt/svyazcom/sbin/replica_loop.sh restart
	echo "All servers run!"
  else
 	ln -sfT ./replica_fallback.conf /opt/svyazcom/etc/replica.conf
	sudo -u svyazcom /opt/svyazcom/sbin/replica_loop.sh restart
	echo "Not all servers run!"
fi
