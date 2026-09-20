FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /purchase-api ./cmd/purchase-api \
	&& CGO_ENABLED=0 go build -o /reconciliation-worker ./cmd/reconciliation-worker \
	&& CGO_ENABLED=0 go build -o /mock-broker ./cmd/mock-broker

FROM alpine:3.21
COPY --from=build /purchase-api /reconciliation-worker /mock-broker /usr/local/bin/
EXPOSE 8080
CMD ["purchase-api"]
