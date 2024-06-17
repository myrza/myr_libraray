FROM  golang:1.22-alpine3.20

WORKDIR /app 

COPY . .

#Download and install dependencies:
RUN go get -d -v ./...

# Build the go app
RUN go build -o api .

EXPOSE 8000 

CMD ["./api"]