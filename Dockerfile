FROM golang:1.23.0

ENV GIN_MODE release

WORKDIR /go/src/app

RUN go install github.com/air-verse/air@latest

COPY ./app .

CMD air