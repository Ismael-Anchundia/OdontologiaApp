FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/dentalia-api .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 app
COPY --from=build /out/dentalia-api /usr/local/bin/dentalia-api
USER app
ENV APP_ENV=production PORT=10000
EXPOSE 10000
CMD ["dentalia-api"]
