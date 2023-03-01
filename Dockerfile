FROM golang:1.17-alpine AS builder

WORKDIR /app
COPY . ./
COPY go.mod ./
COPY go.sum ./
COPY cmd ./
RUN go build -o /app/clubhouse

WORKDIR /dist
RUN cp /app/clubhouse .

FROM scratch
COPY --from=builder /dist/clubhouse .
EXPOSE 50051
ENTRYPOINT ["/clubhouse"]
