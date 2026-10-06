FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -o /bookstore-api ./cmd/api

FROM alpine:3.20
COPY --from=build /bookstore-api /bookstore-api
EXPOSE 8080
ENTRYPOINT ["/bookstore-api"]
