#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(dirname $(readlink -f $0))

pwsh $SCRIPT_DIR/e2e/bin/Debug/net10.0/playwright.ps1 install chromium

if ! command -v certutil &>/dev/null; then
    sudo apt-get update
    sudo apt-get install -y libnss3-tools
fi

#dotnet dev-certs https --trust

sudo mkdir -p /usr/local/share/ca-certificates/aspnet/
dotnet dev-certs https --export-path /tmp/https.crt --format PEM && \
    sudo mv /tmp/https.crt /usr/local/share/ca-certificates/aspnet/

sudo update-ca-certificates

NSS_DB_DIR="${HOME}/.pki/nssdb"
NSS_DB="sql:${NSS_DB_DIR}"
CERT_NAME="ASP.NET Core localhost development certificate"

mkdir -p "${NSS_DB_DIR}"

if ! certutil -L -d "${NSS_DB}" &>/dev/null; then
    certutil -d "${NSS_DB}" -N --empty-password
fi

if certutil -L -d "${NSS_DB}" -n "${CERT_NAME}" &>/dev/null; then
    certutil -D -d "${NSS_DB}" -n "${CERT_NAME}"
fi

certutil -d "${NSS_DB}" -A -t "P,," \
    -n "${CERT_NAME}" \
    -i /usr/local/share/ca-certificates/aspnet/https.crt
