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

Or test the S3 upload:

```shell
curl -v -X PUT http://localhost:9100/s3 \
  -H 'Content-Type: application/json' \
  -d '{
    "bucket": "layer0-dvn-dev",
    "key": "test.txt",
    "contentType": "text/plain",
    "data": "aGVsbG8gd29ybGQ="
  }'
```

This should create a plain text file with `hello world`.

## Set up

Build the executable for Linux amd64: ```make build/linux/amd64```.

Then bake into an EC2 instance with an IAM role with permissions for secretsmanager.
Also, make sure the proper AWS region is set, example:

```shell
mkdir -p ~/.aws
echo "[default]
region = eu-west-2" > ~/.aws/config
```

Then, set up the proxy server as a service:

1. Create a directory for the program `sudo mkdir -p /opt/aws-vsock-proxy` and copy the executable into it with the name
   `server` and ensure it's executable
2. Create a systemd service file `sudo vi /etc/systemd/system/aws-vsock-proxy.service`
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
        systemctl kill -s HUP aws-vsock-proxy.service
    endscript
}
```

And set up log filtering:

```shell
sudo mkdir -p /etc/systemd/journald.conf.d
sudo vi /etc/systemd/journald.conf.d/aws-vsock-proxy.conf
```

```
[Journal]
ForwardToFile=/var/log/journal/aws-vsock-proxy.log
```

Then you can try starting the service and checking the status:

```shell
# Reload systemd to recognize the new service
sudo systemctl daemon-reload
# Reset the journal service
sudo systemctl restart systemd-journald
# Enable the service to start on boot
sudo systemctl enable aws-vsock-proxy.service
# Start the service
sudo systemctl start aws-vsock-proxy.service
# Check the status
sudo systemctl status aws-vsock-proxy.service
```

### Useful commands

```shell
# View logs
sudo journalctl -u aws-vsock-proxy.service

# View recent logs
sudo journalctl -u aws-vsock-proxy.service -f

# Restart service
sudo systemctl restart aws-vsock-proxy.service

# Stop service
sudo systemctl stop aws-vsock-proxy.service
```

### Testing on the server

Once the daemon is running, run a socat proxy temporarily and try curling:

```shell
socat TCP-LISTEN:9100,fork,reuseaddr VSOCK-CONNECT:3:9100
curl -v http://localhost:9100/secret?secretId=dev%2Fgasolina
```
