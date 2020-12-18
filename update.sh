# This updates Go to the latest version
set -ex

# todo: allow to pass in version
# todo: grab latest version automatically

tar="go1.15.6.linux-amd64.tar.gz"
url="https://golang.org/dl/$tar"
echo "url $url"
$(curl -L $url --output go.tar.gz)
$(sudo rm -rf /usr/local/go)
$(sudo tar -C /usr/local -xzf go.tar.gz)
$(rm go.tar.gz)
echo $(go version)
