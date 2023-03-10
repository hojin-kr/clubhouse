FROM golang:1.17-alpine AS builder

WORKDIR /app
COPY . ./
COPY go.mod ./
COPY go.sum ./
COPY cmd ./
RUN go build -o /app/clubhouse
# FROM scratch
# COPY --from=builder /app/clubhouse /app/clubhouse
ENTRYPOINT [ "/app/clubhouse" ]