---
title: "A blog that ships only static files"
date: 2026-09-15T09:00:00-03:00
draft: false
description: "Hugo, PaperMod, Docker, and Chroma in one deliberately small blog."
tags: ["hugo", "docker", "go"]
categories: ["tooling"]
---

This site is compiled by Hugo and served as plain files. The authoring workflow is equally plain: write Markdown, review the generated page, and commit it.

## Go

Chroma understands common backend and frontend languages during the Hugo build.

```go {filename="main.go",linenos=inline,hl_lines=[7],linenostart=1}
package main

import "fmt"

func main() {
	message := "build once, serve anywhere"
	fmt.Println(message)
}
```

## A small service

```python {filename="health.py",linenos=inline}
from http.server import BaseHTTPRequestHandler, HTTPServer

class Health(BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200)
        self.end_headers()
        self.wfile.write(b"ok")

HTTPServer(("0.0.0.0", 8080), Health).serve_forever()
```

## Container configuration

```dockerfile {filename="Dockerfile",linenos=inline}
FROM nginx:alpine
COPY public/ /usr/share/nginx/html/
```

```yaml {filename="compose.yaml",linenos=inline}
services:
  blog:
    image: developer-blog:latest
    ports:
      - "8080:80"
```

```java {filename="Post.java",linenos=inline}
record Post(String title, boolean published) {}
```

No runtime database and no browser-side highlighting bundle are involved.
