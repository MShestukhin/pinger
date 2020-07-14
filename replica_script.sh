#!/usr/bin/env bash
IFS=, read -a ip <<< $1
if "${ip[0]}" = true && "${ip[1]}" = true; then
	echo "All serv"
  else
	echo "Not All serv"
fi