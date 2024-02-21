FROM bufbuild/buf AS bufdownload
RUN buf build buf.build/pentops/o5 -o pentops-o5.binpb

FROM golang:1.22 AS builder

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

COPY --from=bufdownload pentops-o5.binpb /pentops-o5.binpb

ENV MIGRATIONS_DIR=/migrations

CMD ["serve"]
ENTRYPOINT ["/server"]
