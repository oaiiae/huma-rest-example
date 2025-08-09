FROM golang AS build

WORKDIR /app
COPY go.sum go.mod vendor/ ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o huma-rest-example-server -ldflags "-s -w"

# ---

FROM scratch

COPY --from=build /app/huma-rest-example-server /
ENTRYPOINT [ "/huma-rest-example-server" ]
