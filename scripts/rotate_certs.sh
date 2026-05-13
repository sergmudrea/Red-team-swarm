#!/usr/bin/env bash
# rotate_certs.sh – renew self-signed proxy certificates and restart nginx
set -euo pipefail

DOMAIN="${1:?Usage: rotate_certs.sh <domain>}"
CERT_DIR="/etc/nginx/certs"
DAYS=365

# create backup
mkdir -p "${CERT_DIR}/backup"
cp "${CERT_DIR}/proxy.crt" "${CERT_DIR}/backup/proxy-$(date +%Y%m%d).crt" 2>/dev/null || true
cp "${CERT_DIR}/proxy.key" "${CERT_DIR}/backup/proxy-$(date +%Y%m%d).key" 2>/dev/null || true

# generate new key and certificate
openssl req -x509 -nodes -days "${DAYS}" -newkey rsa:2048 \
    -keyout "${CERT_DIR}/proxy.key" \
    -out "${CERT_DIR}/proxy.crt" \
    -subj "/CN=${DOMAIN}"

# restart nginx
systemctl restart nginx
echo "Certificates rotated for ${DOMAIN}"
```Полный путь: `scripts/destroy_proxy.sh`

```bash
#!/usr/bin/env bash
# destroy_proxy.sh – purge nginx, certificates, and logs from a fronting proxy
set -euo pipefail

echo "WARNING: This will completely remove nginx and all related data."
read -p "Continue (yes/no): " confirm
if [ "${confirm}" != "yes" ]; then
    echo "Aborted."
    exit 1
fi

systemctl stop nginx || true
apt-get purge -y nginx nginx-common nginx-full
rm -rf /etc/nginx /var/log/nginx /var/www/hive-cover
echo "Proxy destroyed."
