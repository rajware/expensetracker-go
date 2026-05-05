FROM golang:1.25.9-alpine3.23 AS builder
ARG VERSION_STRING

WORKDIR /app
RUN apk add --no-cache make

COPY go.* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o out/tracker-web -ldflags "-X main.version=${VERSION_STRING}" ./cmd/tracker-web

RUN chmod 755 out/tracker-web \
    && mkdir -p /app/data \
    && chown root:0 /app/data \
    && chmod g+w /app/data \
    && touch /app/data/.keep \
    && chmod g+w /app/data/.keep

FROM alpine:latest AS final
LABEL maintainer="Rajware Services Pvt. Ltd. <support@rajware.net>"
WORKDIR /app

COPY --from=builder /app/out/tracker-web .
COPY --from=builder --chown=0:0 --chmod=775 /app/data /app/data

ENTRYPOINT ["./tracker-web"]
EXPOSE 8080
