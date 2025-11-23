FROM golang:1.21-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /prsvc ./cmd/prsvc

FROM scratch
COPY --from=build /prsvc /prsvc
EXPOSE 8080
ENTRYPOINT ["/prsvc"]
