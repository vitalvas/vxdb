---
title: API
layout: home
---

## Endpoints

Buckets are automatically created when a key is created.

| Path | Method | Description |
| --- | --- | --- |
| / | GET | List of buckets |
| /`<bucket>` | GET | List of keys |
| /`<bucket>` | POST | Create key with unique name <br>(the link will be passed in the http header `location`) |
| /`<bucket>`/`<key name>` | PUT | Create or update data in key |
| /`<bucket>`/`<key name>` | DELETE | Delete key |

### Add-ons

There are some additions to the methods as well.

When viewing a list of buckets or keys, you can filter by prefix using `x-key-prefix` in the request header or `prefix` in the request argument.

When recording, you can specify the TTL for recording. To do this, you need to pass `x-key-ttl` in the header or `ttl` in the request argument.
The time is taken as a value, reflected in seconds and in the format of an integer-positive number.

## Management

| Path | Method | Description |
| --- | --- | --- |
| /metrics | GET | Prometheus metrics |
| /api/backup | GET | Export KV via protobuf file |
| /api/restore | PUT | Import KV from protobuf file |
