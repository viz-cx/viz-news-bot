FROM golang:latest as builder
RUN mkdir -p /go/src/github.com/viz-cx/viz-news-bot
WORKDIR /go/src/github.com/viz-cx/viz-news-bot
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a --ldflags '-extldflags "-static"' -o bin/viz-news-bot -i .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder [ \
    "/go/src/github.com/viz-cx/viz-news-bot/bin/viz-news-bot", \
    "/go/src/github.com/viz-cx/viz-news-bot/.env", \
    "./"]
ENTRYPOINT ["./viz-news-bot"]
