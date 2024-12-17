#!/usr/bin/env bash
JOB_NAME=$1
NAMESPACE=$2
IDENTITY_FILE=$3
REMOTE_USER=$4
EXPECTED_ROOT_SIZE=$5
EXPECTED_DATA_SIZE=$6

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

function check_size() {
    local BLK_DEVICES=$1
    local VOLUME=$2
    local EXPECTED_SIZE=$3

    SIZE=$(echo "${BLK_DEVICES}" | jq .blockdevices | jq -r --arg name "${VOLUME}" '.[] | select(.name == $name) | .size')
    if [[ $SIZE == "${EXPECTED_SIZE}" ]]; then
        return 0
    fi
    return 1
}

function check_sizes() {
    local BLK_DEVICES=$1

    if ! check_size "${BLK_DEVICES}" vda "${EXPECTED_ROOT_SIZE}"; then
        return 1
    fi

    local DATA_VOLUMES=("vdb" "vdc")
    for vol in "${DATA_VOLUMES[@]}"; do
        if ! check_size "${BLK_DEVICES}" "${vol}" "${EXPECTED_DATA_SIZE}"; then
            return 1
        fi
    done

    return 0
}

VMS=$(get_vms "${NAMESPACE}" "${JOB_NAME}")

for vm in ${VMS}; do
    for attempt in $(seq 1 $MAX_RETRIES); do
        BLK_DEVICES=$(remote_command "${NAMESPACE}" "${IDENTITY_FILE}" "${REMOTE_USER}" "${vm}" "lsblk --json -v --output=NAME,SIZE")
        RET=$?
        if [ $RET -eq 0 ] && check_sizes "${BLK_DEVICES}"; then
            break
        fi
        if [ "${attempt}" -lt $MAX_RETRIES ]; then
            sleep $SLEEP_DURATION
        else
            exit 1
        fi
    done
done
