#!/usr/bin/env bash
if [ "$1" = true && "$2" = true]; then
     ln -sft ./replica_fallback.conf /opt/svyazcom/etc/replica.conf
  else
    ln -sft ./replica_backup.conf /opt/svyazcom/etc/replica.conf
fi