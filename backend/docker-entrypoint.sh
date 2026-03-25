#!/bin/sh
set -eu

knowledge_dir="${KNOWLEDGE_STORAGE_PATH:-/var/lib/trustmesh-knowledge}"
trustmesh_uid="$(id -u trustmesh)"
trustmesh_gid="$(id -g trustmesh)"

mkdir -p "${knowledge_dir}"

current_owner="$(stat -c '%u:%g' "${knowledge_dir}")"
expected_owner="${trustmesh_uid}:${trustmesh_gid}"

if [ "${current_owner}" != "${expected_owner}" ]; then
  chown -R trustmesh:trustmesh "${knowledge_dir}"
fi

exec su-exec trustmesh "$@"
