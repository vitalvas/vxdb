---
title: Configuration
layout: home
---

Applications are configured through environment variables.

Configuration parameters:

| Key | Description | Default value |
| -- | -- | -- |
| DB_PATH | Path to database files | `/data` (for docker)<br>`/var/lib/vxdb` (for packages) |
| HTTP_HOST | Bind address of REST interface | `0.0.0.0:8080` |
| ENCRYPTION_KEY | Data encryption key | |
| AUTH_DATA_JWKS_URL | Jwks url for auth using JWT keys | |

### Encryption

The encryption is based on AES.
The key must be 16, 24 or 32 bytes and packed in base64.

Type of AES is used based on the key size. For example 16 bytes will use AES-128. 24 bytes will use AES-192. 32 bytes will use AES-256.

You can use openssl to generate the key:

* `openssl rand -base64 16`
* `openssl rand -base64 24`
* `openssl rand -base64 32`
