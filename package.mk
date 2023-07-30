ifndef PKR_VAR_clair_version
	PKR_VAR_clair_version = $(shell grep Version debian/control | cut -d' ' -f2)
endif

/tmp/build-clair/var/lib/clair/GIT_REV:
	mkdir -p /tmp/build-clair
	mkdir -p /tmp/build-clair/usr/share/bash-completion/completions
	mkdir -p /tmp/build-clair/usr/bin
	mkdir -p /tmp/build-clair/usr/share/doc/clair
	mkdir -p /tmp/build-clair/usr/share/lintian/overrides
	mkdir -p /tmp/build-clair/usr/share/man/man1
	mkdir -p /tmp/build-clair/var/lib/clair/core-plugins/available

	cp clair /tmp/build-clair/usr/bin
	cp LICENSE /tmp/build-clair/usr/share/doc/clair/copyright
	cp contrib/bash-completion /tmp/build-clair/usr/share/bash-completion/completions/clair
	find . -name ".DS_Store" -depth -exec rm {} \;
	$(MAKE) go-build
	cp common.mk /tmp/build-clair/var/lib/clair/core-plugins/common.mk
	cp -r plugins/* /tmp/build-clair/var/lib/clair/core-plugins/available
	find plugins/ -mindepth 1 -maxdepth 1 -type d -printf '%f\n' | while read plugin; do cd /tmp/build-clair/var/lib/clair/core-plugins/available/$$plugin && if [ -e Makefile ]; then $(MAKE) src-clean; fi; done
	find plugins/ -mindepth 1 -maxdepth 1 -type d -printf '%f\n' | while read plugin; do touch /tmp/build-clair/var/lib/clair/core-plugins/available/$$plugin/.core; done
	rm /tmp/build-clair/var/lib/clair/core-plugins/common.mk
	$(MAKE) help2man
	$(MAKE) addman
	cp /usr/local/share/man/man1/clair.1 /tmp/build-clair/usr/share/man/man1/clair.1
	gzip -9 /tmp/build-clair/usr/share/man/man1/clair.1
ifeq ($(CLAIR_VERSION),master)
	git describe --tags > /tmp/build-clair/var/lib/clair/VERSION
else
	echo $(CLAIR_VERSION) > /tmp/build-clair/var/lib/clair/VERSION
endif
ifdef CLAIR_GIT_REV
	echo "$(CLAIR_GIT_REV)" > /tmp/build-clair/var/lib/clair/GIT_REV
else
	git rev-parse HEAD > /tmp/build-clair/var/lib/clair/GIT_REV
endif

.PHONY: image/build/digitalocean
image/build/digitalocean:
	packer build -var 'clair_version=${PKR_VAR_clair_version}' contrib/images/digitalocean/packer.pkr.hcl

.PHONY: image/init/digitalocean
image/init/digitalocean:
	packer init contrib/images/digitalocean/packer.pkr.hcl

.PHONY: image/validate/digitalocean
image/validate/digitalocean:
	packer validate -var 'clair_version=${PKR_VAR_clair_version}' contrib/images/digitalocean/packer.pkr.hcl
