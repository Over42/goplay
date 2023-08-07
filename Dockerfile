FROM golang:1.20
WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 GOARCH=amd64 go build -o bin/goplay main.go

EXPOSE 8080
CMD ["bin/goplay"]
