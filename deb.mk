BUILD_DIRECTORY    ?= /tmp
CLAIR_DESCRIPTION   = 'Docker powered PaaS that helps you build and manage the lifecycle of applications'
CLAIR_REPO_NAME    ?= clair/clair
CLAIR_ARCHITECTURE ?= amd64

ifndef IS_RELEASE
	IS_RELEASE = true
endif


.PHONY: install-from-deb deb-all deb-clair deb-setup

install-from-deb:
	@echo "--> Initial apt-get update"
	sudo apt-get update -qq >/dev/null
	sudo apt-get -qq -y --no-install-recommends install apt-transport-https

	@echo "--> Installing docker"
	wget -nv -O - https://get.docker.com/ | sh

	@echo "--> Installing clair"
	wget -qO- https://packagecloud.io/clair/clair/gpgkey | sudo tee /etc/apt/trusted.gpg.d/clair.asc
	@echo "deb https://packagecloud.io/clair/clair/ubuntu/ $(shell lsb_release -cs 2>/dev/null || echo "focal") main" | sudo tee /etc/apt/sources.list.d/clair.list
	sudo apt-get update -qq >/dev/null
	sudo DEBIAN_FRONTEND=noninteractive DEBCONF_NONINTERACTIVE_SEEN=true apt-get -qq -y --no-install-recommends install clair

deb-all: deb-setup deb-clair
	mv $(BUILD_DIRECTORY)/*.deb .
	@echo "Done"

deb-setup:
	@echo "-> Updating deb repository and installing build requirements"
	@sudo apt-get update -qq >/dev/null
	@sudo DEBIAN_FRONTEND=noninteractive DEBCONF_NONINTERACTIVE_SEEN=true apt-get -qq -y --no-install-recommends install gcc git build-essential wget ruby-dev ruby1.9.1 lintian >/dev/null 2>&1
	@command -v fpm >/dev/null || sudo gem install fpm --no-ri --no-rdoc
	@ssh -o StrictHostKeyChecking=no git@github.com || true

deb-clair: /tmp/build-clair/var/lib/clair/GIT_REV
	rm -f $(BUILD_DIRECTORY)/clair_*_$(CLAIR_ARCHITECTURE).deb

	cat /tmp/build-clair/var/lib/clair/VERSION | cut -d '-' -f 1 | cut -d 'v' -f 2 > /tmp/build-clair/var/lib/clair/STABLE_VERSION
ifneq (,$(findstring false,$(IS_RELEASE)))
	sed -i.bak -e "s/^/`date +%s`:/" /tmp/build-clair/var/lib/clair/STABLE_VERSION && rm /tmp/build-clair/var/lib/clair/STABLE_VERSION.bak
endif

	cp -r debian /tmp/build-clair/DEBIAN
	sed -i.bak "s/^Architecture: .*/Architecture: $(CLAIR_ARCHITECTURE)/g" /tmp/build-clair/DEBIAN/control && rm  /tmp/build-clair/DEBIAN/control.bak
	rm -f /tmp/build-clair/DEBIAN/lintian-overrides
	cp debian/lintian-overrides /tmp/build-clair/usr/share/lintian/overrides/clair
	sed -i.bak "s/^Version: .*/Version: `cat /tmp/build-clair/var/lib/clair/STABLE_VERSION`/g" /tmp/build-clair/DEBIAN/control && rm /tmp/build-clair/DEBIAN/control.bak
	dpkg-deb --build /tmp/build-clair "$(BUILD_DIRECTORY)/clair_`cat /tmp/build-clair/var/lib/clair/VERSION`_$(CLAIR_ARCHITECTURE).deb"
	lintian "$(BUILD_DIRECTORY)/clair_`cat /tmp/build-clair/var/lib/clair/VERSION`_$(CLAIR_ARCHITECTURE).deb" || true
