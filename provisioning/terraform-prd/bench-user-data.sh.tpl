#!/usr/bin/env bash

cat << _EOF_ >> /home/isucon/isuxportal-supervisor.env
ISUXPORTAL_SUPERVISOR_INSTANCE_NAME=$(curl -s --retry 5 --retry-connrefused --max-time 10 --connect-timeout 5 http://169.254.169.254/latest/meta-data/local-ipv4)
ISUXPORTAL_SUPERVISOR_ENDPOINT_URL=https://isuxportal-prd-grpc.xi.isucon.dev
ISUXPORTAL_SUPERVISOR_TOKEN=${isuxportal_supervisor_token}
ISUXPORTAL_SUPERVISOR_TEAM_ID=${isuxportal_supervisor_team_id}
_EOF_
chown isucon: /home/isucon/isuxportal-supervisor.env
