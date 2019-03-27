FROM golang:1.11.5-alpine

WORKDIR /go/src/github.com/Trirandom/capstone-server
COPY . .
RUN GO111MODULE=off CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o capstone .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0  /go/src/github.com/Trirandom/capstone-server/capstone .
CMD ["./capstone"]