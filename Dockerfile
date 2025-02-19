FROM golang:1.22.5 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o final-project ./main.go

FROM ubuntu:latest

WORKDIR /app

COPY --from=builder /app/go-final-project /app/go-final-project

COPY web /app/web

COPY database/scheduler.db /app/scheduler.db

COPY variable.env /app/variable.env

ENV TODO_PORT=7540
ENV TODO_DBFILE=/app/scheduler.db

EXPOSE 7540

CMD ["/app/go-final-project"]
