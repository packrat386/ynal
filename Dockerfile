FROM golang:latest as builder

WORKDIR /ynal
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -o /go/bin/ynal

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /go/bin/ynal /usr/local/bin

WORKDIR /ynal
ENTRYPOINT ["ynal"]