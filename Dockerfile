ARG EXPORTER_VER=0.1.0

FROM golang:alpine3.12

WORKDIR /build
ADD go.mod go.sum ./
RUN go mod download

ARG EXPORTER_VER
ADD . ./
RUN go build \
        -v \
        -ldflags="-w -s -X 'main.Version=$EXPORTER_VER'" \
        -o /nzbget_exporter

# ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

FROM alpine:3.12

ARG EXPORTER_VER

LABEL maintainer="frebib <nzbget-exporter@frebib.net>" \
      org.label-schema.vendor="frebib" \
      org.label-schema.name="nzbget-exporter" \
      org.label-schema.url="https://github.com/frebib/nzbget-exporter" \
      org.label-schema.description="NZBGet Prometheus metrics exporter" \
      org.label-schema.version=${EXPORTER_VER}

COPY --from=0 /nzbget_exporter /usr/bin
CMD ["/usr/bin/nzbget_exporter"]
