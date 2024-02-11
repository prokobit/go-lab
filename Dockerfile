FROM golang:1.22 AS build

WORKDIR /app

COPY . ./
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bin/hello ./cmd/hello/main.go

FROM gcr.io/distroless/static-debian12
COPY --from=build /app/bin/hello /
ENTRYPOINT ["/hello"]