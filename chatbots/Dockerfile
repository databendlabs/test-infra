FROM golang:1.15 AS builder
WORKDIR app
COPY . .

RUN mkdir -p bin && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/bot ./chatbots/cmd/... && mv ./bin/bot /tmp/bot && chmod a+x /tmp/bot

FROM debian:buster
RUN apt-get update && apt-get install ca-certificates -y
COPY ./chatbots/deploy/DigiCertGlobalRootCA.crt /usr/local/share/ca-certificates/ca.crt
RUN update-ca-certificates --fresh > /dev/null
COPY --from=builder /tmp/bot /bot
CMD ["/bot"]
