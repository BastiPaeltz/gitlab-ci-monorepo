FROM golang:alpine AS build-env
ADD . /src
RUN cd /src && go build -o /tmp/gitlab-trigger-proxy

FROM alpine
WORKDIR /app
COPY --from=build-env /tmp/gitlab-trigger-proxy /app/
ENTRYPOINT ["./gitlab-trigger-proxy"]
CMD ["--help"]
