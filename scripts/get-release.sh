#!/bin/bash

set -euo pipefail

wget https://github.com/fatcatfablab/doorbot2/releases/download/${1}/doorbot2-linux-armv7.tar.gz
tar -zxf doorbot2-linux-armv7.tar.gz
rm *.tar.gz

chmod o-rx *
mv doorbot2-linux-armv7 ../doorbot2
mv doorbot2.service /etc/systemd/system/

systemctl daemon-reload
systemctl restart doorbot2
systemctl status doorbot2