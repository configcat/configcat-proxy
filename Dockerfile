FROM golang:1.24-alpine3.21 AS build

WORKDIR /go/src/configcat_proxy

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -o /go/bin/configcat_proxy -ldflags "-s -w" .

FROM gcr.io/distroless/static-debian11

COPY --from=build --chown=nonroot:nonroot /go/bin/configcat_proxy /

USER nonroot
EXPOSE 8050 8051 50051

ENTRYPOINT ["/configcat_proxy"]