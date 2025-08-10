FROM golang:1.24-alpine AS builder

WORKDIR ${GOPATH}/src/github.com/sams96/bookmark-feeder/

COPY go.mod go.sum ${GOPATH}/src/github.com/sams96/bookmark-feeder/
RUN go mod download

COPY . ${GOPATH}/src/github.com/sams96/bookmark-feeder/

RUN go build -o /go/bin/bookmark-feeder .

FROM docker

ADD https://github.com/golang/go/raw/master/lib/time/zoneinfo.zip /zoneinfo.zip
ENV ZONEINFO=/zoneinfo.zip

COPY --from=builder /go/bin/bookmark-feeder /usr/bin/bookmark-feeder

ENTRYPOINT ["/usr/bin/bookmark-feeder"]
