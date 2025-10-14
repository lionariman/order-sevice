# build stage
FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/app ./cmd/app

# run stage
FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /app/app /app/app
COPY .env.example /app/.env
COPY web /app/web
EXPOSE 8081
ENTRYPOINT ["/app/app"]
