FROM golang:1.19.6-alpine3.17 as builder

WORKDIR /go/src/configcat_proxy

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -o /go/bin/configcat_proxy .

FROM alpine:3.17

RUN addgroup -g 1001 -S configcat_group && \
    adduser -u 1001 -S configcat_user -G configcat_group

RUN apk add --no-cache ca-certificates \
    libcrypto1.1 \
    libssl1.1

RUN update-ca-certificates

COPY --from=builder /go/bin/configcat_proxy /go/bin/configcat_proxy

USER 1001
EXPOSE 8050 8051 50051

ENTRYPOINT ["/go/bin/configcat_proxy"]