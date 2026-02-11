BEAT_NAME=netatmobeat
BEAT_PATH=github.com/radoondas/netatmobeat
ES_BEATS_MODULE=github.com/elastic/beats/v7
ES_BEATS=$(shell go list -m -f '{{.Dir}}' $(ES_BEATS_MODULE) 2>/dev/null || { go mod download $(ES_BEATS_MODULE) >&2 && go list -m -f '{{.Dir}}' $(ES_BEATS_MODULE); })
LIBBEAT_MAKEFILE=$(ES_BEATS)/libbeat/scripts/Makefile
SYSTEM_TESTS=false
TEST_ENVIRONMENT=false
NO_COLLECT=true
CHECK_HEADERS_DISABLED=true
BEAT_VENDOR=radoondas
BEAT_LICENSE=ASL 2.0
GOBUILD_FLAGS=-ldflags "-X $(ES_BEATS_MODULE)/libbeat/version.buildTime=$(NOW) -X $(ES_BEATS_MODULE)/libbeat/version.commit=$(COMMIT_ID)"

-include $(LIBBEAT_MAKEFILE)
