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
