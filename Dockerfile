FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY main.go .
RUN go build -o server main.go

FROM alpine:3.20
COPY --from=builder /app/server /server
ENV PORT=8080
EXPOSE 8080
CMD ["/server"]