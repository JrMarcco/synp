FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN go mod download

COPY . .

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o synp ./cmd/main.go

FROM golang:1.25-alpine

WORKDIR /app

COPY --from=builder /app/synp .
# 配置文件
# COPY --from=builder /app/etc/config.yaml config.yaml

RUN chmod +x ./synp

CMD ["./synp", "--config=config.yaml"]
