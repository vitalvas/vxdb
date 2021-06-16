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
