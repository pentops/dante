FROM golang:1.21 AS builder

RUN mkdir /src
WORKDIR /src


COPY go.mod go.sum .
RUN --mount=type=cache,target=/go/pkg/mod \
	go mod download -x

COPY . .

ARG VERSION

RUN \
	--mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	CGO_ENABLED=0 go build -ldflags="-X main.Version=$VERSION" -v -o /server ./cmd/server/

FROM scratch

LABEL org.opencontainers.image.source=https://github.com/pentops/dante

COPY --from=builder /server /server
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /src/ext/db /migrations
ENV MIGRATIONS_DIR=/migrations

CMD ["serve"]
ENTRYPOINT ["/server"]