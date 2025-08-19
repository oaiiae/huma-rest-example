FROM gcr.io/distroless/static-debian12:nonroot
COPY huma-rest-example /
ENTRYPOINT ["/huma-rest-example"]
