FROM 528451384384.dkr.ecr.us-west-2.amazonaws.com/segment-golang:1.17.6

COPY . /go/src/github.com/segmentio/stats

ENV CGO_ENABLED=0
RUN apk add --no-cache git && \
    cd /go/src/github.com/segmentio/stats && \
    go build -v -o /dogstatsd ./cmd/dogstatsd && \
    apk del git && \
    rm -rf /go/*

ENTRYPOINT ["/dogstatsd"]
