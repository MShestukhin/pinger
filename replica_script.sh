#!/usr/bin/env bash
IFS=, read -a ip <<< $1
if "${ip[0]}" = true && "${ip[1]}" = true; then
  ln -sf ./replica_backup.conf /opt/svyazcom/etc/replica.conf
	echo "All servers run!"
  else
  ln -sf ./replica_fallback.conf /opt/svyazcom/etc/replica.conf
	echo "Not all servers run!"
fi