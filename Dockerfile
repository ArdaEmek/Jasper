FROM golang:1.27.3-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main cmd/api/main.go
RUN CGO_ENABLED=0 go install github.com/amacneil/dbmate/v2@v2.27.0

FROM alpine:3.23.4 AS prod
WORKDIR /app
COPY --from=build /app/main /app/main
COPY --from=build /go/bin/dbmate /usr/local/bin/dbmate
COPY --from=build /app/db/migrations /app/db/migrations
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh
ENV PORT=8080 \
    STORAGE_DIR=/app/data \
    HTTP3_ENABLED=true
EXPOSE 8080
ENTRYPOINT ["./docker-entrypoint.sh"]
CMD ["serve"]
