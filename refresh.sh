#!/bin/sh
cd "$(dirname "$0")"
./sitemanager
rcctl restart httpd
