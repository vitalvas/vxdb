---
title: Example
layout: home
---

## Docker

```yml
version: '3'

services:
  vxdb:
    image: vitalvas/vxdb:latest
    ports:
      - '8080:8080'
    volumes:
      - <data dir>:/data
```

### Gitlab data auth

#### Compose

```yml
version: '3'

services:
  vxdb:
    image: vitalvas/vxdb:latest
    ports:
      - '8080:8080'
    volumes:
      - <data dir>:/data
    environment:
      AUTH_DATA_JWKS_URL: 'https://gitlab.com/-/jwks'
```

#### CI Job

```shell
curl  -H "Authorization: Bearer ${CI_JOB_JWT}" http://vxdb.service.local:8080/test_bucket/test_key
```
