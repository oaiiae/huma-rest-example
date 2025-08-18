FROM golang AS build

ARG BUILD_NAME=huma-rest-example
ARG BUILD_VERSION
ARG BUILD_DATE

WORKDIR /app
COPY go.sum go.mod vendor/ ./
COPY . .
RUN export BUILD_VERSION=${BUILD_VERSION:-git-$(git rev-parse --short HEAD)} && \
    export BUILD_DATE=${BUILD_DATE:-$(date -Is)} && \
    export CGO_ENABLED=0 GOOS=linux GOARCH=amd64 && \
    go build -o bin/${BUILD_NAME} -ldflags "-s -w -X main.BuildName=${BUILD_NAME} -X main.BuildVersion=${BUILD_VERSION} -X main.BuildDate=${BUILD_DATE}"
RUN cd bin && ln -s ${BUILD_NAME} entrypoint

# ---

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /app/bin /
ENTRYPOINT [ "/entrypoint" ]
