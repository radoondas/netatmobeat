FROM golang:1.24 AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o netatmobeat .

FROM ubuntu:24.04

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/* && \
    groupadd --system netatmobeat && \
    useradd --system --no-create-home --gid netatmobeat netatmobeat

COPY --from=builder /build/netatmobeat /usr/share/netatmobeat/netatmobeat
COPY netatmobeat.docker.yml /usr/share/netatmobeat/netatmobeat.yml
COPY netatmobeat.reference.yml /usr/share/netatmobeat/netatmobeat.reference.yml
COPY netatmobeat.template.publicdata.json /usr/share/netatmobeat/
COPY netatmobeat.template.stastiondata.json /usr/share/netatmobeat/

RUN chmod 755 /usr/share/netatmobeat/netatmobeat

USER netatmobeat

ENTRYPOINT ["/usr/share/netatmobeat/netatmobeat"]
CMD ["-c", "/usr/share/netatmobeat/netatmobeat.yml", "-e"]