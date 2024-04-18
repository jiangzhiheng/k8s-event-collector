FROM golang:1.21 AS build
ENV GOPROXY=https://proxy.golang.org
WORKDIR /go/utils/k8s-event-collector
COPY . .
RUN CGO_ENABLED=0 go build -o /go/bin/k8s-event-collector cmd/main.go

FROM centos:latest
MAINTAINER "1689991551@qq.com"
COPY --from=build /go/bin/* /utils/

WORKDIR /utils
ENTRYPOINT ["./k8s-event-collector"]