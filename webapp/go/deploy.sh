#!/bin/bash
make || exit

APPNAME=isucholar

SERVER=
# SERVER=
# SERVER=

ssh $SERVER << EOF
sudo systemctl stop ${APPNAME}.go.service
EOF

sftp $SERVER <<EOF
cd /home/isucon/webapp/go
put ${APPNAME}
EOF

ssh $SERVER << EOF
sudo systemctl start  ${APPNAME}.go.service
sudo systemctl status ${APPNAME}.go.service
isulog lotate
EOF
