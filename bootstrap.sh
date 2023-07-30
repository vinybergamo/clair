#!/usr/bin/env bash
set -eo pipefail
[[ $TRACE ]] && set -x

# A script to bootstrap clair.
# It expects to be run on Ubuntu 20.04/22.04 via 'sudo`
# If installing a tag higher than 0.3.13, it may install clair via a package (so long as the package is higher than 0.3.13)
# It checks out the clair source code from GitHub into ~/clair and then runs 'make install' from clair source.

# We wrap this whole script in functions, so that we won't execute
# until the entire script is downloaded.
# That's good because it prevents our output overlapping with wget's.
# It also means that we can't run a partially downloaded script.

SUPPORTED_VERSIONS="Debian [10, 11, 12], Ubuntu [18.04, 20.04, 22.04]"

log-fail() {
  declare desc="log fail formatter"
  echo "$@" 1>&2
  exit 1
}

ensure-environment() {
  local FREE_MEMORY
  if [[ -z "$CLAIR_TAG" ]]; then
    echo "Preparing to install $CLAIR_REPO..."
  else
    echo "Preparing to install $CLAIR_TAG from $CLAIR_REPO..."
  fi

  hostname -f >/dev/null 2>&1 || {
    log-fail "This installation script requires that you have a hostname set for the instance. Please set a hostname for 127.0.0.1 in your /etc/hosts"
  }

  FREE_MEMORY=$(grep MemTotal /proc/meminfo | awk '{print $2}')
  if [[ "$FREE_MEMORY" -lt 1003600 ]]; then
    echo "For clair to build containers, it is strongly suggested that you have 1024 megabytes or more of free memory"
    echo "If necessary, please consult this document to setup swap: https://clair.com/docs/getting-started/advanced-installation/#vms-with-less-than-1-gb-of-memory"
  fi
}

install-requirements() {
  echo "--> Ensuring we have the proper dependencies"

  case "$CLAIR_DISTRO" in
    debian)
      if ! dpkg -l | grep -q gpg-agent; then
        apt-get update -qq >/dev/null
        apt-get -qq -y --no-install-recommends install gpg-agent
      fi
      if ! dpkg -l | grep -q software-properties-common; then
        apt-get update -qq >/dev/null
        apt-get -qq -y --no-install-recommends install software-properties-common
      fi
      ;;
    ubuntu)
      if ! dpkg -l | grep -q gpg-agent; then
        apt-get update -qq >/dev/null
        apt-get -qq -y --no-install-recommends install gpg-agent
      fi
      if ! dpkg -l | grep -q software-properties-common; then
        apt-get update -qq >/dev/null
        apt-get -qq -y --no-install-recommends install software-properties-common
      fi

      add-apt-repository -y universe >/dev/null
      apt-get update -qq >/dev/null
      ;;
  esac
}

install-clair() {
  if ! command -v clair &>/dev/null; then
    echo "--> Note: Installing clair for the first time will result in removal of"
    echo "    files in the nginx 'sites-enabled' directory. Please manually"
    echo "    restore any files that may be removed after the installation and"
    echo "    web setup is complete."
    echo ""
    echo "    Installation will continue in 10 seconds."
    sleep 10
  fi

  if [[ -n $CLAIR_BRANCH ]]; then
    install-clair-from-source "origin/$CLAIR_BRANCH"
  elif [[ -n $CLAIR_TAG ]]; then
    local CLAIR_SEMVER="${CLAIR_TAG//v/}"
    major=$(echo "$CLAIR_SEMVER" | awk '{split($0,a,"."); print a[1]}')
    minor=$(echo "$CLAIR_SEMVER" | awk '{split($0,a,"."); print a[2]}')
    patch=$(echo "$CLAIR_SEMVER" | awk '{split($0,a,"."); print a[3]}')

    use_plugin=false
    # 0.4.0 implemented a `plugin` plugin
    if [[ "$major" -eq "0" ]] && [[ "$minor" -ge "4" ]] && [[ "$patch" -ge "0" ]]; then
      use_plugin=true
    elif [[ "$major" -ge "1" ]]; then
      use_plugin=true
    fi

    # 0.3.13 was the first version with a debian package
    if [[ "$major" -eq "0" ]] && [[ "$minor" -eq "3" ]] && [[ "$patch" -ge "13" ]]; then
      install-clair-from-package "$CLAIR_SEMVER"
      echo "--> Running post-install dependency installation"
      clair plugins-install-dependencies
    elif [[ "$use_plugin" == "true" ]]; then
      install-clair-from-package "$CLAIR_SEMVER"
      echo "--> Running post-install dependency installation"
      sudo -E clair plugin:install-dependencies --core
    else
      install-clair-from-source "$CLAIR_TAG"
    fi
  else
    install-clair-from-package
    echo "--> Running post-install dependency installation"
    sudo -E clair plugin:install-dependencies --core
  fi
}

install-clair-from-source() {
  local CLAIR_CHECKOUT="$1"

  if ! command -v apt-get &>/dev/null; then
    log-fail "This installation script requires apt-get. For manual installation instructions, consult https://clair.com/docs/getting-started/advanced-installation/"
  fi

  apt-get -qq -y --no-install-recommends install sudo git make software-properties-common
  cd /root
  if [[ ! -d /root/clair ]]; then
    git clone "$CLAIR_REPO" /root/clair
  fi

  cd /root/clair
  git fetch origin
  [[ -n $CLAIR_CHECKOUT ]] && git checkout "$CLAIR_CHECKOUT"
  make install
}

install-clair-from-package() {
  case "$CLAIR_DISTRO" in
    debian | ubuntu)
      install-clair-from-deb-package "$@"
      ;;
    *)
      log-fail "Unsupported Linux distribution. For manual installation instructions, consult https://clair.com/docs/getting-started/advanced-installation/"
      ;;
  esac
}

in-array() {
  declare desc="return true if value ($1) is in list (all other arguments)"

  local e
  for e in "${@:2}"; do
    [[ "$e" == "$1" ]] && return 0
  done
  return 1
}

install-clair-from-deb-package() {
  local CLAIR_CHECKOUT="$1"
  local NO_INSTALL_RECOMMENDS=${CLAIR_NO_INSTALL_RECOMMENDS:=""}
  local OS_ID

  if ! in-array "$CLAIR_DISTRO_VERSION" "18.04" "20.04" "22.04" "10" "11" "12"; then
    log-fail "Unsupported Linux distribution. Only the following versions are supported: $SUPPORTED_VERSIONS"
  fi

  if [[ -n $CLAIR_DOCKERFILE ]]; then
    NO_INSTALL_RECOMMENDS=" --no-install-recommends "
  fi

  echo "--> Initial apt-get update"
  apt-get update -qq >/dev/null
  apt-get -qq -y --no-install-recommends install apt-transport-https

  if ! command -v docker &>/dev/null; then
    echo "--> Installing docker"
    if uname -r | grep -q linode; then
      echo "--> NOTE: Using Linode? Docker may complain about missing AUFS support."
      echo "    You can safely ignore this warning."
      echo ""
      echo "    Installation will continue in 10 seconds."
      sleep 10
    fi
    export CHANNEL=stable
    wget -nv -O - https://get.docker.com/ | sh
  fi

  OS_ID="$(lsb_release -cs 2>/dev/null || echo "bionic")"
  if ! in-array "$CLAIR_DISTRO" "debian" "ubuntu" "raspbian"; then
    CLAIR_DISTRO="ubuntu"
    OS_ID="bionic"
  fi

  if [[ "$CLAIR_DISTRO" == "ubuntu" ]]; then
    OS_IDS=("bionic" "focal" "jammy")
    if ! in-array "$OS_ID" "${OS_IDS[@]}"; then
      OS_ID="bionic"
    fi
  elif [[ "$CLAIR_DISTRO" == "debian" ]]; then
    OS_IDS=("stretch" "buster" "bullseye" "bookworm")
    if ! in-array "$OS_ID" "${OS_IDS[@]}"; then
      OS_ID="bullseye"
    fi
  elif [[ "$CLAIR_DISTRO" == "raspbian" ]]; then
    OS_IDS=("buster" "bullseye")
    if ! in-array "$OS_ID" "${OS_IDS[@]}"; then
      OS_ID="bullseye"
    fi
  fi

  echo "--> Installing clair"
  wget -qO- https://packagecloud.io/clair/clair/gpgkey | sudo tee /etc/apt/trusted.gpg.d/clair.asc
  echo "deb https://packagecloud.io/clair/clair/$CLAIR_DISTRO/ $OS_ID main" | tee /etc/apt/sources.list.d/clair.list
  apt-get update -qq >/dev/null

  [[ -n $CLAIR_VHOST_ENABLE ]] && echo "clair clair/vhost_enable boolean $CLAIR_VHOST_ENABLE" | sudo debconf-set-selections
  [[ -n $CLAIR_HOSTNAME ]] && echo "clair clair/hostname string $CLAIR_HOSTNAME" | sudo debconf-set-selections
  [[ -n $CLAIR_SKIP_KEY_FILE ]] && echo "clair clair/skip_key_file boolean $CLAIR_SKIP_KEY_FILE" | sudo debconf-set-selections
  [[ -n $CLAIR_KEY_FILE ]] && echo "clair clair/key_file string $CLAIR_KEY_FILE" | sudo debconf-set-selections
  [[ -n $CLAIR_NGINX_ENABLE ]] && echo "clair clair/nginx_enable string $CLAIR_NGINX_ENABLE" | sudo debconf-set-selections

  if [[ -n $CLAIR_CHECKOUT ]]; then
    # shellcheck disable=SC2086
    apt-get -qq -y $NO_INSTALL_RECOMMENDS install "clair=$CLAIR_CHECKOUT"
  else
    # shellcheck disable=SC2086
    apt-get -qq -y $NO_INSTALL_RECOMMENDS install clair
  fi
}

main() {
  export CLAIR_DISTRO CLAIR_DISTRO_VERSION
  # shellcheck disable=SC1091
  CLAIR_DISTRO=$(. /etc/os-release && echo "$ID")
  # shellcheck disable=SC1091
  CLAIR_DISTRO_VERSION=$(. /etc/os-release && echo "$VERSION_ID")

  export DEBIAN_FRONTEND=noninteractive
  export CLAIR_REPO=${CLAIR_REPO:-"https://github.com/clair/clair.git"}

  ensure-environment
  install-requirements
  install-clair

  if [[ -f /etc/update-motd.d/99-clair ]]; then
    /etc/update-motd.d/99-clair || true
  fi
}

main "$@"
