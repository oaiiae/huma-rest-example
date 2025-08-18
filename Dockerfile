FROM golang AS build

ARG BUILD_NAME=huma-rest-example
ARG BUILD_VERSION=dev
ARG BUILD_DATE=""

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

WORKDIR /app
COPY go.sum go.mod vendor/ ./
COPY . .
RUN go build -o bin/${BUILD_NAME} -ldflags "-s -w -X main.BuildName=${BUILD_NAME} -X main.BuildVersion=${BUILD_VERSION} -X main.BuildDate=${BUILD_DATE}"
RUN cd bin && ln -s ${BUILD_NAME} entrypoint

# ---

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /app/bin /
ENTRYPOINT [ "/entrypoint" ]
