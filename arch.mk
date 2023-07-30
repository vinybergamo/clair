.PHONY: arch-all arch-clair

arch-all: arch-clair
	echo "Done"

arch-setup:
	echo "-> Updating pacman repository and installing package helper"
	sudo pacman -Sy
	sudo pacman -S --needed --noconfirm pkgbuild-introspection

	echo "-> Download, build and install cower (dependency of pacaur) and pacaur"
	curl -so /tmp/cower.tar.gz https://aur.archlinux.org/cgit/aur.git/snapshot/cower.tar.gz
	curl -so /tmp/pacaur.tar.gz https://aur.archlinux.org/cgit/aur.git/snapshot/pacaur.tar.gz
	tar -xzf /tmp/cower.tar.gz -C /tmp
	tar -xzf /tmp/pacaur.tar.gz -C /tmp
	gpg --recv-key 1EB2638FF56C0C53
	cd /tmp/cower; makepkg -sri --noconfirm
	cd /tmp/pacaur; makepkg -sri --noconfirm

	echo "-> Installing build requirements"
	pacaur --noconfirm --noedit -S plugn

arch-clair: arch-setup
	echo "-> Update package sums, create metadata file and test the build of the package"
ifeq ($(CLAIR_VERSION),master)
	git describe --tags > /tmp/VERSION
else
	echo $(CLAIR_VERSION) > /tmp/VERSION
endif
	cat /tmp/VERSION | cut -d '-' -f 1 | cut -d 'v' -f 2 > /tmp/STABLE_VERSION
	sed -i -e "s/pkgver=.*/pkgver=`cat /tmp/STABLE_VERSION`/" /clair-arch/PKGBUILD
	cd /clair-arch; updpkgsums; mksrcinfo; makepkg -fd
