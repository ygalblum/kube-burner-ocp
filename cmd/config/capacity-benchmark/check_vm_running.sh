#!/usr/bin/env bash
JOB_NAME=$1
NAMESPACE=$2
IDENTITY_FILE=$3
REMOTE_USER=$4

MAX_RETRIES=10
SLEEP_DURATION=30

function get_vms() {
    local namespace=$1
    local job_name=$2

    local vms=$(kubectl get vm -n "${namespace}" -l kube-burner-job="${job_name}" -o json | jq .items | jq -r '.[] | .metadata.name')
    local ret=$?
    if [ $ret -ne 0 ]; then
        echo "Failed to get VM list"
        exit 1
    fi
    echo $vms
}

function remote_command() {
    local namespace=$1
    local identity_file=$2
    local remote_user=$3
    local vm_name=$4
    local command=$5

    local output
    output=$(virtctl ssh --local-ssh-opts="-o StrictHostKeyChecking=no"  --local-ssh-opts="-o UserKnownHostsFile=/dev/null" -n "${namespace}" -i "${identity_file}" -c "${command}" "${remote_user}"@"${vm_name}")
    local ret=$?
    if [ $ret -ne 0 ]; then
        return 1
    fi
    echo $output
}

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