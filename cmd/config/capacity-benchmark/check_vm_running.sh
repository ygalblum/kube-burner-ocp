#!/usr/bin/env bash
JOB_NAME=$1
NAMESPACE=$2
IDENTITY_FILE=$3
REMOTE_USER=$4

source helpers.bash

VMS=$(get_vms "${NAMESPACE}" "${JOB_NAME}")

for vm in ${VMS}; do
    for attempt in $(seq 1 $MAX_RETRIES); do
        if remote_command "${NAMESPACE}" "${IDENTITY_FILE}" "${REMOTE_USER}" "${vm}" "ls"; then
            break
        fi
        if [ "${attempt}" -lt $MAX_RETRIES ]; then
            sleep $SLEEP_DURATION
        else
            exit 1
        fi
    done
done
