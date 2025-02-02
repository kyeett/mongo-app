FROM golang:1.16.3 as build-env

# All these steps will be cached
RUN mkdir /app
WORKDIR /app
COPY go.mod .
COPY go.sum .

# Get dependancies - will also be cached if we won't change mod/sum
RUN go mod download

# COPY the source code as the last step
COPY main.go .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -installsuffix cgo -o /go/bin/app

#######

FROM scratch
COPY --from=build-env /go/bin/app /go/bin/app
ENTRYPOINT ["/go/bin/app"]