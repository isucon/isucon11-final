#!/bin/bash

mkdir -p ~isucon/.ssh
curl --retry 5 --retry-connrefused --max-time 10 --connect-timeout 5 "https://portal.isucon.net/api/ssh_public_keys?token=${checker_token}" > ~isucon/.ssh/authorized_keys
chmod 0700 ~isucon/.ssh
chmod 0600 ~isucon/.ssh/authorized_keys
chown -R isucon:isucon ~isucon/.ssh
