FROM golang:1.15.5

ENV GOPROXY="https://goproxy.cn,direct"
WORKDIR /go/src/github.com/heptiolabs/eventrouter

ADD eventrouter.go .
ADD main.go .
ADD sinks sinks
ADD go.mod .
RUN pwd
RUN go build -o main main.go eventrouter.go

FROM ubuntu:18.04
COPY --from=0 /go/src/github.com/heptiolabs/eventrouter/main /main
ENTRYPOINT ["/main"]