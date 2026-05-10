FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY main.go .
RUN go build -o server main.go

FROM alpine:3.20
RUN addgroup -S app && adduser -S -G app -u 1000 app
COPY --from=builder --chown=app:app /app/server /server
USER 1000
ENV PORT=8080
EXPOSE 8080
CMD ["/server"]
