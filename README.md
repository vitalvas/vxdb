# VxDB

Simple Key-Value NoSQL database with simplest API interface.

Perfect for serverless apps, prototyping, metrics, and more.

## Usage

### Env vars

- `DB_PATH` - path to DB files
- `HTTP_HOST` - listen http host and port (default: `0.0.0.0:8080`)

### Endpoints

Buckets are automatically created when a key is created.

| Path | Method | Description |
| --- | --- | --- |
| GET | / | List of buckets |
| GET | /`<bucket>` | List of keys |
| POST | /`<bucket>` | Create key with unique name (the link will be passed in the http header `location`) |
| PUT | /`<bucket>`/`<key name>` | Create or update data in key |
| DELETE | /`<bucket>`/`<key name>` | Delete key |

### With docker

```yml
version: '3'
services:
  vxdb: vitalvas/vxdb:latest
  ports:
    - '8080:8080'
  volumes:
    - <data dir>:/data
  environment:
    DB_PATH: '/data'
```
