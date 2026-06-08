FROM golang:1.24-alpine AS build

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/erp-server ./cmd/server

FROM alpine:3.20

WORKDIR /app
COPY --from=build /bin/erp-server /usr/local/bin/erp-server

EXPOSE 8080
ENV APP_ADDR=:8080

CMD ["erp-server"]
