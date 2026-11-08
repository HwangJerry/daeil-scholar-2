#!/bin/bash
# Renew the mobile IP certificate and apply it after validating Apache config.
set -euo pipefail
umask 077

readonly CERTIFICATE=/etc/alumni-ip-tls/certificates/211.169.249.240.crt
readonly LEGO=/usr/local/bin/lego-ip
readonly MINIMUM_VALIDITY_SECONDS=172800

exec 9>/run/lock/alumni-ip-certificate.lock
flock -n 9 || exit 0

"$LEGO" run --accept-tos --account-id alumni-ip \
    --path /etc/alumni-ip-tls --server letsencrypt \
    --domains 211.169.249.240 --profile shortlived --key-type RSA2048 \
    --http --http.webroot /var/lib/alumni-ip-acme/webroot \
    --renew-days 3 --no-random-sleep

# Always retry applying the current certificate, including after a prior reload
# failure. A deploy hook alone would not retry until the next successful renewal.
/usr/sbin/httpd -t
/usr/bin/systemctl reload httpd
/usr/bin/openssl x509 -checkend "$MINIMUM_VALIDITY_SECONDS" -noout -in "$CERTIFICATE"
/usr/local/bin/alumni-ip-https-check
