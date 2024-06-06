FROM golang:latest as builder

MAINTAINER astra<wuyusanhua2023@gmail.com>

ENV GO111MODULE=on\
    GOPROXY=https://goproxy.cn,direct\
    CGO_ENABLED=0\
    GOOS=linux\
    GOARCH=amd64

WORKDIR /app/webhook

COPY ./src .

RUN go mod tidy && \
    go build -o main .

FROM alpine

COPY --from=builder /app/webhook /

EXPOSE 9094

CMD ["/main"]