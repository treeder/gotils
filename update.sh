# This updates Go to the latest version
set -ex

# copied from: https://gist.github.com/davivcgarcia/2fea719c67f1c6282bc53df46f7add25#file-update-golang-sh Thanks!

# Checks if is running as root, and sudo if not
[ `whoami` = root ] || { sudo "$0" "$@"; exit $?; }

# Determines current local version
if [[ -f /usr/local/go/bin/go ]]; then
    CURRENT=$(/usr/local/go/bin/go version | grep -oP "go\d+\.\d+(\.\d+)?")
else
    CURRENT=""
fi

# Determine latest available version
LATEST=$(curl -sL https://godoc.org/golang.org/dl | grep -oP "go\d+\.\d+(\.\d+)?\s" | awk '{$1=$1};1' | sort -V | uniq | tail -n 1)

# Checks if update is required
if [[ ${CURRENT} == "${LATEST}" ]]; then
    echo "System is already up to date."
    exit 0
else
    echo "Updating to version ${LATEST}:"

    # Downloads latest tarball
    curl -# https://dl.google.com/go/${LATEST}.linux-amd64.tar.gz -o /usr/local/${LATEST}.linux-amd64.tar.gz

    # Remove old installation
    rm -rf /usr/local/go

    # Unpack tarball
    tar -C /usr/local -xzf /usr/local/${LATEST}.linux-amd64.tar.gz

    # Remove tarball
    rm -rf /usr/local/${LATEST}.linux-amd64.tar.gz

    echo "Done!"
    exit 0
fi
