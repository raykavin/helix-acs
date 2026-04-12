#!/bin/bash
set -e

log()  { echo "> $*"; }
info() { echo "  $*"; }
err()  { echo "[✘] $*"; exit 1; }

log "Setup Go Private Proxy - $GO_PROXY_HOST"

# DEPENDENCIES
log "Installing dependencies..."
sudo apt-get install -y nginx dnsutils openssl > /dev/null 2>&1
info "nginx, dnsutils, openssl installed"

# RESOLVE IP
log "Resolving IP for $GO_PROXY_HOST..."
PROXY_IP=$(dig +short "$GO_PROXY_HOST" | grep -E '^[0-9]+\.' | head -1)
[ -z "$PROXY_IP" ] && err "Could not resolve IP for $GO_PROXY_HOST"
info "IP resolved: $PROXY_IP"

# SELF-SIGNED CERTIFICATE
log "Generating self-signed certificate..."
sudo openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout /etc/ssl/private/go-proxy-local.key \
  -out /etc/ssl/certs/go-proxy-local.crt \
  -subj "/CN=$GO_PROXY_HOST" \
  -addext "subjectAltName=DNS:$GO_PROXY_HOST" > /dev/null 2>&1
info "Certificate generated"

# TRUST SELF-SIGNED CERTIFICATE
log "Trusting self-signed certificate..."
sudo cp /etc/ssl/certs/go-proxy-local.crt /usr/local/share/ca-certificates/go-proxy-local.crt
sudo update-ca-certificates > /dev/null 2>&1
info "Certificate trusted"

# NGINX CONFIGURATION
log "Configuring nginx..."
sudo tee /etc/nginx/sites-available/go-proxy > /dev/null << EOF
server {
    listen 443 ssl;
    server_name $GO_PROXY_HOST;

    ssl_certificate     /etc/ssl/certs/go-proxy-local.crt;
    ssl_certificate_key /etc/ssl/private/go-proxy-local.key;

    location / {
        proxy_pass            https://$PROXY_IP:$GO_PROXY_PORT;
        proxy_ssl_server_name on;
        proxy_ssl_name        $GO_PROXY_HOST;
        proxy_set_header      Host $GO_PROXY_HOST;
        proxy_set_header      Accept-Encoding "";

        # Strip port from go-import meta-tag so Go can resolve the module path
        sub_filter            '$GO_PROXY_HOST:$GO_PROXY_PORT' '$GO_PROXY_HOST';
        sub_filter_once       off;
    }
}
EOF

sudo rm -f /etc/nginx/sites-enabled/default
sudo ln -sf /etc/nginx/sites-available/go-proxy /etc/nginx/sites-enabled/go-proxy
sudo nginx -t > /dev/null 2>&1 || err "Invalid nginx configuration"
sudo service nginx restart > /dev/null 2>&1
info "nginx configured and started"

# HOSTS FILE
log "Configuring /etc/hosts..."
if grep -q "$GO_PROXY_HOST" /etc/hosts; then
  info "$GO_PROXY_HOST already in /etc/hosts, replacing..."
  sudo sed -i "/$GO_PROXY_HOST/d" /etc/hosts
fi
echo "127.0.0.1 $GO_PROXY_HOST" | sudo tee -a /etc/hosts > /dev/null
info "/etc/hosts updated"

# GIT AUTHENTICATION
log "Configuring git authentication..."
git config --global url."https://$GIT_CONFIG_DEV_USERNAME:$GIT_CONFIG_TOKEN@$GO_PROXY_HOST/".insteadOf "https://$GO_PROXY_HOST/"
info "Git credentials configured"

# GO PRIVATE ENVIRONMENT
log "Configuring Go private env vars..."
go env -w GOPRIVATE="$GO_PROXY_HOST/*"
go env -w GONOSUMDB="$GO_PROXY_HOST/*"
go env -w GONOPROXY="$GO_PROXY_HOST/*"
info "GOPRIVATE, GONOSUMDB, GONOPROXY set"

log "Proxy setup completed"
