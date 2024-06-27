FROM golang:1.22.1 AS build
WORKDIR /src
COPY . /src/
RUN  cd cmd/aigogo; CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -v -o /bin/app main.go

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /bin/app /bin/app
EXPOSE 8080
CMD ["/bin/app"]
