FROM golang:1.18 AS build
WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY main/*.go ./
RUN go build -o /raft-example
EXPOSE 8080
EXPOSE 9021
CMD ["/raft-example"]