FROM golang:1.25-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates git
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/bquick-secret-api ./cmd/bquick-secret-api

FROM alpine:3.20
RUN adduser -D -H bquicksecret
USER bquicksecret
COPY --from=build /out/bquick-secret-api /usr/local/bin/bquick-secret-api
EXPOSE 8081
ENTRYPOINT ["bquick-secret-api"]
