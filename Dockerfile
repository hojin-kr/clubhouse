FROM golang:1.17-alpine

WORKDIR /app
COPY . ./
COPY go.mod ./
COPY go.sum ./
COPY cmd ./
RUN go build -o /app/clubhouse

# ENV PORT=50051
# EXPOSE 50051

CMD [ "/app/clubhouse" ]
