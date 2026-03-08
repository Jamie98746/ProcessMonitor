# syntax=docker/dockerfile:1

# Build stage
FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /procmon ./cmd

# Final stage
FROM scratch

COPY --from=build /procmon /procmon

ENTRYPOINT ["/procmon"]
