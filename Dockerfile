FROM golang AS builder

COPY . /app
WORKDIR /app
RUN CGO_ENABLED=0 go build -o bingo ./cmd/server

FROM scratch

COPY --from=builder /app/bingo /app/bingo
COPY groups /app/groups
COPY tasks /app/tasks
COPY web /app/web

EXPOSE 8080/tcp

WORKDIR /app

CMD ["./bingo"]
