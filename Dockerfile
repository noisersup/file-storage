FROM golang:1.17.7-alpine
WORKDIR /app

COPY . /app
RUN go mod download && go mod verify

RUN go build -o fileStorage main.go

CMD ["fileStorage"]
