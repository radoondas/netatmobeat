BEAT_NAME=netatmobeat
BEAT_PATH=github.com/radoondas/netatmobeat
BEAT_GOPATH=$(firstword $(subst :, ,${GOPATH}))
SYSTEM_TESTS=false
TEST_ENVIRONMENT=false
ES_BEATS?=./vendor/github.com/elastic/beats
LIBBEAT_MAKEFILE=$(ES_BEATS)/libbeat/scripts/Makefile
GOPACKAGES=$(shell govendor list -no-status +local)
GOBUILD_FLAGS=-i -ldflags "-X $(BEAT_PATH)/vendor/github.com/elastic/beats/libbeat/version.buildTime=$(NOW) -X $(BEAT_PATH)/vendor/github.com/elastic/beats/libbeat/version.commit=$(COMMIT_ID)"
MAGE_IMPORT_PATH=${BEAT_PATH}/vendor/github.com/magefile/mage
NO_COLLECT=true
CHECK_HEADERS_DISABLED=true
BEAT_VENDOR=radoondas
BEAT_LICENSE=ASL 2.0
VIRTUALENV_PARAMS=-p /usr/bin/python2.7

# Path to the libbeat Makefile
-include $(LIBBEAT_MAKEFILE)

.PHONY: copy-vendor
copy-vendor:
	mage vendorUpdate
