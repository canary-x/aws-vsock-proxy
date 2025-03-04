# AWS vsock Proxy

Implements API via vsock, exposing basic AWS operations, like getting secrets from secretmanager.
It is meant to run on an EC2 instance with Nitro installed, to allow Nitro enclaves to access AWS services with ease.

## Testing locally

Assume the canary nonprod AWS profile and start the server:

```shell
export AWS_PROFILE=canary-nonprod
go run github.com/canary-x/aws-vsock-proxy/cmd/server
```

Then test the secret retrieval:
```shell
curl -v http://localhost:9100/secret?secretId=dev%2Fgasolina
```

## Set up

Build the executable for Linux amd64: ```make build/linux/amd64```.

Then bake into an EC2 instance and set it up as a service:

1. Create a directory for the program `sudo mkdir -p /opt/aws-vsock-proxy` and copy the executable into it
2. Create a systemd service file `sudo vi /etc/systemd/system/myapp.service`
3. Add the following content:

```
[Unit]
Description=AWS VSOCK proxy
After=network.target

[Service]
Type=simple
User=ec2-user
Group=ec2-user
WorkingDirectory=/opt/aws-vsock-proxy
ExecStart=/opt/aws-vsock-proxy/server
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

Also set up log rotation by creating `sudo vi /etc/logrotate.d/aws-vsock-proxy` with the following config:
```
/var/log/journal/aws-vsock-proxy.log {
    daily
    rotate 14
    compress
    delaycompress
    missingok
    notifempty
    create 0640 ec2-user ec2-user
    postrotate
        systemctl kill -s HUP myapp.service
    endscript
}
```

And set up log filtering:
```shell
sudo mkdir -p /etc/systemd/journald.conf.d
sudo vi /etc/systemd/journald.conf.d/myapp.conf
```

```
[Journal]
ForwardToFile=/var/log/journal/myapp.log
```

Then you can try starting the service and checking the status:
```shell
# Reload systemd to recognize the new service
sudo systemctl daemon-reload
# Reset the journal service
sudo systemctl restart systemd-journald
# Enable the service to start on boot
sudo systemctl enable myapp.service
# Start the service
sudo systemctl start myapp.service
# Check the status
sudo systemctl status myapp.service
```

### Useful commands

```shell
# View logs
sudo journalctl -u myapp.service

# View recent logs
sudo journalctl -u myapp.service -f

# Restart service
sudo systemctl restart myapp.service

# Stop service
sudo systemctl stop myapp.service
```

### Testing on the server

Once the daemon is running, run a socat proxy temporarily and try curling: 
```shell
socat TCP-LISTEN:9100,fork,reuseaddr VSOCK-CONNECT:3:9100
curl -v http://localhost:9100/secret?secretId=dev%2Fgasolina
```
