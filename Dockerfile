# syntax=docker/dockerfile:1

ARG GO_VERSION=1.27.1
ARG HUGO_VERSION=0.134.3
ARG PAPERMOD_VERSION=v8.0

# Pin the Go 1.27 toolchain, then install the matching official Hugo Extended binary.
FROM golang:${GO_VERSION}-bookworm AS hugo
ARG HUGO_VERSION
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && \
    rm -rf /var/lib/apt/lists/* && \
    curl --fail --location --retry 3 \
      --output /tmp/hugo.tar.gz \
      "https://github.com/gohugoio/hugo/releases/download/v${HUGO_VERSION}/hugo_extended_${HUGO_VERSION}_linux-amd64.tar.gz" && \
    tar -C /usr/local/bin -xzf /tmp/hugo.tar.gz hugo && \
    hugo version

# Keep the theme outside the source bind mount used by Docker Compose.
FROM alpine:3.22 AS theme
ARG PAPERMOD_VERSION
RUN apk add --no-cache ca-certificates git && \
    git clone --depth 1 --branch "${PAPERMOD_VERSION}" \
      https://github.com/adityatelange/hugo-PaperMod.git /themes/PaperMod

FROM hugo AS development
WORKDIR /src
COPY --from=theme /themes/PaperMod /opt/themes/PaperMod
EXPOSE 1313
ENTRYPOINT ["hugo"]
CMD ["server", "--bind", "0.0.0.0", "--baseURL", "http://localhost:1313/", "--themesDir", "/opt/themes", "--buildDrafts", "--disableFastRender"]

FROM hugo AS site-builder
ARG HUGO_BASEURL=https://blog.example.com/
WORKDIR /src
COPY --from=theme /themes/PaperMod /opt/themes/PaperMod
COPY . .
RUN hugo --gc --minify --themesDir /opt/themes --baseURL "${HUGO_BASEURL}"

FROM nginx:1.29-alpine AS production
COPY nginx/default.conf /etc/nginx/conf.d/default.conf
COPY --from=site-builder /src/public /usr/share/nginx/html
EXPOSE 80
