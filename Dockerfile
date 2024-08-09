FROM golang:1.22-alpine as builder
ARG LDFLAGS

RUN apk --no-cache add ca-certificates \
    && rm -Rf /var/cache/apk/*

WORKDIR /workspace
COPY go.mod go.sum /workspace/
RUN go mod download

COPY cmd /workspace/cmd
COPY pkg /workspace/pkg
COPY main.go /workspace/
RUN CGO_ENABLED=0 go build -a -ldflags "${LDFLAGS}" -o aliyun_exporter main.go && ./aliyun_exporter --version

FROM alpine:3

RUN addgroup -S appusers && adduser -S appuser -G appusers
USER appuser
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /workspace/aliyun_exporter /usr/local/bin/aliyun_exporter

ENTRYPOINT ["/usr/local/bin/aliyun_exporter"]
