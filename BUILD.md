# Netatmobeat

Welcome to Netatmobeat.

## Getting Started with Netatmobeat

### Requirements

* [Go](https://golang.org/dl/) 1.24+

### Clone

```
git clone https://github.com/radoondas/netatmobeat
cd netatmobeat
```

### Build

To build the binary for Netatmobeat run the command below. This will generate a binary
in the same directory with the name netatmobeat.

```
go build ./...
```

### Run

To run Netatmobeat with debugging output enabled, run:

```
./netatmobeat -c netatmobeat.yml -e -d "*"
```

### Test

```
go test ./beater/...
go test ./config/...
go vet ./...
```

### Update

Each beat has a template for the mapping in Elasticsearch and a documentation for the fields
which is automatically generated based on `fields.yml`.
To generate templates and field docs:

```
make update
```

### Cleanup

To format source code:

```
make fmt
```

To clean up the build directory and generated artifacts:

```
make clean
```

For further development, check out the [beat developer guide](https://www.elastic.co/guide/en/beats/libbeat/current/new-beat.html).

## Packaging

The beat framework provides tools to cross-compile and package your beat for different platforms. This requires [docker](https://www.docker.com/). To build packages:

```
mage package
```

This will fetch and create all images required for the build process. The whole process to finish can take several minutes.
