CLAIR_VERSION ?= master

TARGETARCH ?= amd64

DOCKER_IMAGE_LABELER_VERSION ?= 0.1.0
HEROKUISH_VERSION ?= 0.6.0
LAMBDA_BUILDER_VERSION ?= 0.4.0
NETRC_VERSION ?= 0.6.0
PLUGN_VERSION ?= 0.12.0
PROCFILE_VERSION ?= 0.15.0
SIGIL_VERSION ?= 0.9.0
SSHCOMMAND_VERSION ?= 0.16.0
DOCKER_IMAGE_LABELER_URL ?= https://github.com/vinybergamo/docker-image-labeler/releases/download/v${DOCKER_IMAGE_LABELER_VERSION}/docker-image-labeler_${DOCKER_IMAGE_LABELER_VERSION}_linux_${TARGETARCH}.tgz
LAMBDA_BUILDER_URL ?= https://github.com/vinybergamo/lambda-builder/releases/download/v${LAMBDA_BUILDER_VERSION}/lambda-builder_${LAMBDA_BUILDER_VERSION}_linux_${TARGETARCH}.tgz
NETRC_URL ?= https://github.com/vinybergamo/netrc/releases/download/v${NETRC_VERSION}/netrc_${NETRC_VERSION}_linux_${TARGETARCH}.tgz
PLUGN_URL ?= https://github.com/vinybergamo/plugn/releases/download/v${PLUGN_VERSION}/plugn_${PLUGN_VERSION}_linux_${TARGETARCH}.tgz
PROCFILE_UTIL_URL ?= https://github.com/josegonzalez/go-procfile-util/releases/download/v${PROCFILE_VERSION}/procfile-util_${PROCFILE_VERSION}_linux_${TARGETARCH}.tgz
SIGIL_URL ?= https://github.com/gliderlabs/sigil/releases/download/v${SIGIL_VERSION}/gliderlabs-sigil_${SIGIL_VERSION}_linux_${TARGETARCH}.tgz
SSHCOMMAND_URL ?= https://github.com/vinybergamo/sshcommand/releases/download/v${SSHCOMMAND_VERSION}/sshcommand_${SSHCOMMAND_VERSION}_linux_x86_64.tgz
STACK_URL ?= https://github.com/gliderlabs/herokuish.git
PREBUILT_STACK_URL ?= gliderlabs/herokuish:latest-20
CLAIR_LIB_ROOT ?= /var/lib/clair
PLUGINS_PATH ?= ${CLAIR_LIB_ROOT}/plugins
CORE_PLUGINS_PATH ?= ${CLAIR_LIB_ROOT}/core-plugins
PLUGIN_MAKE_TARGET ?= build-in-docker

# If the first argument is "vagrant-clair"...
ifeq (vagrant-clair,$(firstword $(MAKECMDGOALS)))
  # use the rest as arguments for "vagrant-clair"
  RUN_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  # ...and turn them into do-nothing targets
  $(eval $(RUN_ARGS):;@:)
endif

ifeq ($(CIRCLECI),true)
	BUILD_STACK_TARGETS = circleci deps build
else
	BUILD_STACK_TARGETS = build-in-docker
endif

include common.mk

.PHONY: all apt-update install version copyfiles copyplugin man-db plugins dependencies docker-image-labeler lambda-builder netrc sshcommand procfile-util plugn docker aufs stack count vagrant-acl-add vagrant-clair go-build

include tests.mk
include package.mk
include deb.mk
include arch.mk

all:
	# Type "make install" to install.

install: dependencies version copyfiles plugin-dependencies plugins


packer:
	packer build contrib/packer.json

go-build:
	basedir=$(PWD); \
	for dir in plugins/*; do \
		if [ -e $$dir/Makefile ]; then \
			$(MAKE) -e -C $$dir $(PLUGIN_MAKE_TARGET) || exit $$? ;\
		fi ;\
	done


go-build-plugin:
ifndef PLUGIN_NAME
	$(error PLUGIN_NAME not specified)
endif
	if [ -e plugins/$(PLUGIN_NAME)/Makefile ]; then \
		$(MAKE) -e -C plugins/$(PLUGIN_NAME) $(PLUGIN_MAKE_TARGET) || exit $$? ;\
	fi

go-clean:
	basedir=$(PWD); \
	for dir in plugins/*; do \
		if [ -e $$dir/Makefile ]; then \
			$(MAKE) -e -C $$dir clean ;\
		fi ;\
	done

copyfiles:
	$(MAKE) go-build || exit 1
	cp clair /usr/local/bin/clair
	mkdir -p ${CORE_PLUGINS_PATH} ${PLUGINS_PATH}
	rm -rf ${CORE_PLUGINS_PATH}/*
	test -d ${CORE_PLUGINS_PATH}/enabled || PLUGIN_PATH=${CORE_PLUGINS_PATH} plugn init
	test -d ${PLUGINS_PATH}/enabled || PLUGIN_PATH=${PLUGINS_PATH} plugn init
	find plugins/ -mindepth 1 -maxdepth 1 -type d -printf '%f\n' | while read plugin; do $(MAKE) copyplugin PLUGIN_NAME=$$plugin; done
ifndef SKIP_GO_CLEAN
	$(MAKE) go-clean
endif
	chown clair:clair -R ${PLUGINS_PATH} ${CORE_PLUGINS_PATH} || true
	$(MAKE) addman

copyplugin:
ifndef PLUGIN_NAME
	$(error PLUGIN_NAME not specified)
endif
	rm -Rf ${CORE_PLUGINS_PATH}/available/$(PLUGIN_NAME) && \
		rm -Rf ${PLUGINS_PATH}/available/$(PLUGIN_NAME) && \
		rm -rf ${CORE_PLUGINS_PATH}/$(PLUGIN_NAME) && \
		rm -rf ${PLUGINS_PATH}/$(PLUGIN_NAME) && \
		cp -R plugins/$(PLUGIN_NAME) ${CORE_PLUGINS_PATH}/available && \
		rm -rf ${CORE_PLUGINS_PATH}/available/$(PLUGIN_NAME)/src && \
		ln -s ${CORE_PLUGINS_PATH}/available/$(PLUGIN_NAME) ${PLUGINS_PATH}/available; \
		find /var/lib/clair/ -xtype l -delete;\
		PLUGIN_PATH=${CORE_PLUGINS_PATH} plugn enable $(PLUGIN_NAME) ;\
		PLUGIN_PATH=${PLUGINS_PATH} plugn enable $(PLUGIN_NAME)
	chown clair:clair -R ${PLUGINS_PATH} ${CORE_PLUGINS_PATH} || true

addman: help2man man-db
	mkdir -p /usr/local/share/man/man1
ifneq ("$(wildcard /usr/local/share/man/man1/clair.1-generated)","")
	cp /usr/local/share/man/man1/clair.1-generated /usr/local/share/man/man1/clair.1
else
	help2man -Nh help -v version -n "configure and get information from your clair installation" -o /usr/local/share/man/man1/clair.1 clair
endif
	mandb

version:
	mkdir -p ${CLAIR_LIB_ROOT}
ifeq ($(CLAIR_VERSION),master)
	git describe --tags > ${CLAIR_LIB_ROOT}/VERSION  2>/dev/null || echo '~${CLAIR_VERSION} ($(shell date -uIminutes))' > ${CLAIR_LIB_ROOT}/VERSION
else
	echo $(CLAIR_VERSION) > ${CLAIR_LIB_ROOT}/STABLE_VERSION
endif

plugin-dependencies: plugn procfile-util
	sudo -E clair plugin:install-dependencies --core

plugins: plugn procfile-util docker
	sudo -E clair plugin:install --core

dependencies: apt-update docker-image-labeler lambda-builder netrc sshcommand plugn procfile-util docker help2man man-db sigil dos2unix jq parallel
	$(MAKE) -e stack

apt-update:
	apt-get update -qq

parallel:
	apt-get -qq -y --no-install-recommends install parallel

jq:
	apt-get -qq -y --no-install-recommends install jq

dos2unix:
	apt-get -qq -y --no-install-recommends install dos2unix

help2man:
	apt-get -qq -y --no-install-recommends install help2man

man-db:
	apt-get -qq -y --no-install-recommends install man-db

docker-image-labeler:
	wget -qO /tmp/docker-image-labeler_latest.tgz ${DOCKER_IMAGE_LABELER_URL}
	tar xzf /tmp/docker-image-labeler_latest.tgz -C /usr/local/bin
	mv /usr/local/bin/docker-image-labeler-${TARGETARCH} /usr/local/bin/docker-image-labeler

lambda-builder:
	wget -qO /tmp/lambda-builder_latest.tgz ${LAMBDA_BUILDER_URL}
	tar xzf /tmp/lambda-builder_latest.tgz -C /usr/local/bin
	mv /usr/local/bin/lambda-builder-${TARGETARCH} /usr/local/bin/lambda-builder

netrc:
	wget -qO /tmp/netrc_latest.tgz ${NETRC_URL}
	tar xzf /tmp/netrc_latest.tgz -C /usr/local/bin
	mv /usr/local/bin/netrc-${TARGETARCH} /usr/local/bin/netrc

procfile-util:
	wget -qO /tmp/procfile-util_latest.tgz ${PROCFILE_UTIL_URL}
	tar xzf /tmp/procfile-util_latest.tgz -C /usr/local/bin
	mv /usr/local/bin/procfile-util-${TARGETARCH} /usr/local/bin/procfile-util

plugn:
	wget -qO /tmp/plugn_latest.tgz ${PLUGN_URL}
	tar xzf /tmp/plugn_latest.tgz -C /usr/local/bin
	mv /usr/local/bin/plugn-${TARGETARCH} /usr/local/bin/plugn

sigil:
	wget -qO /tmp/sigil_latest.tgz ${SIGIL_URL}
	tar xzf /tmp/sigil_latest.tgz -C /usr/local/bin
	mv /usr/local/bin/gliderlabs-sigil-${TARGETARCH} /usr/local/bin/sigil

sshcommand:
	wget -qO /tmp/sshcommand_latest.tgz ${SSHCOMMAND_URL}
	tar xzf /tmp/sshcommand_latest.tgz -C /usr/local/bin
	sshcommand create clair /usr/local/bin/clair

docker:
	apt-get -qq -y --no-install-recommends install curl
	grep -i -E "^docker" /etc/group || groupadd docker
	usermod -aG docker clair
ifndef CI
	wget -nv -O - https://get.docker.com/ | sh
ifdef DOCKER_VERSION
	apt-get -qq -y --no-install-recommends install docker-engine=${DOCKER_VERSION} || (apt-cache madison docker-engine ; exit 1)
endif
	sleep 2 # give docker a moment i guess
endif

stack:
ifeq ($(shell test -e /var/run/docker.sock && touch -c /var/run/docker.sock && echo $$?),0)
ifdef BUILD_STACK
	@echo "Start building herokuish from source"
	docker images | grep gliderlabs/herokuish || (git clone ${STACK_URL} /tmp/herokuish && cd /tmp/herokuish && IMAGE_NAME=gliderlabs/herokuish BUILD_TAG=latest VERSION=master make -e ${BUILD_STACK_TARGETS} && rm -rf /tmp/herokuish)
else
ifeq ($(shell echo ${PREBUILT_STACK_URL} | grep -q -E 'http.*://|file://' && echo $$?),0)
	@echo "Start importing herokuish from ${PREBUILT_STACK_URL}"
	docker images | grep gliderlabs/herokuish || wget -nv -O - ${PREBUILT_STACK_URL} | gunzip -cd | docker import - gliderlabs/herokuish
else
	@echo "Start pulling herokuish from ${PREBUILT_STACK_URL}"
	docker images | grep gliderlabs/herokuish || docker pull ${PREBUILT_STACK_URL}
endif
endif
endif

count:
	@echo "Core lines:"
	@cat clair bootstrap.sh | sed 's/^$$//g' | wc -l
	@echo "Plugin lines:"
	@find plugins -type f -not -name .DS_Store | xargs cat | sed 's/^$$//g' | wc -l
	@echo "Test lines:"
	@find tests -type f -not -name .DS_Store | xargs cat | sed 's/^$$//g' | wc -l

vagrant-acl-add:
	vagrant ssh -- sudo sshcommand acl-add clair $(USER)

vagrant-clair:
	vagrant ssh -- "sudo -H -u root bash -c 'clair $(RUN_ARGS)'"
