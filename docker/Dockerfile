FROM golang:1.13 AS builder
COPY . /workdir/
WORKDIR /workdir/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make

FROM alpine
RUN apk add --no-cache ca-certificates
COPY --from=builder /workdir/bin/ssm-env /bin/ssm-env

ENTRYPOINT ["ssm-env"]