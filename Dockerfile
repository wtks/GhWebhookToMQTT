FROM golang:1.11.0-alpine AS build
ENV GO111MODULE=on
RUN apk add --no-cache git
WORKDIR /src
COPY ./go.* ./
RUN go mod download
COPY ./*.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app

# runtime image
FROM alpine:3.8
RUN apk add --no-cache ca-certificates
COPY --from=build /app /app
EXPOSE 8080
ENTRYPOINT ["/app"]
