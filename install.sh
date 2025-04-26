#!/bin/bash
filename=ps-exporter
useradd --no-create-home --shell /bin/false $filename
chmod +x $filename
cp $filename /usr/local/bin

chown $filename:$filename /usr/local/bin/$filename

cat > /etc/systemd/system/$filename.service <<EOF
[Unit]
Description=$filename
Wants=network-online.target
After=network-online.target

[Service]
User=$filename
Group=$filename
Type=simple
ExecStart=/usr/local/bin/$filename -config.file=/etc/$filename/config.yaml
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

chmod 640 /etc/systemd/system/$filename.service

systemctl daemon-reload
systemctl start $filename
systemctl enable $filename