FROM golang:1.21-alpine as build

COPY . /opt/osprober
RUN cd /opt/osprober/; \
    go build -o osprober osprober.go

FROM alpine
COPY --from=build /opt/osprober/osprober /

ENTRYPOINT ["/osprober"]
