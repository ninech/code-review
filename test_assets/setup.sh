#!/bin/bash

# ensure we are in the same dir as the running script
cd `dirname $0`

# newer versions than this time out when stopping...
# But thankfully we don't have a dependency on newer features.
export K8S_VERSION="1.19.2"
export GOOS=$(go env GOOS)
export GOARCH=$(go env GOARCH)

# there's (still) no releases for arm64 on MacOS, so use the amd64
# version. It will be emulated and seems to work fine.
if [[ "$GOARCH" == "arm64" && "$GOOS" == "darwin" ]] ; then
    export GOARCH="amd64"
fi

curl -sSLo envtest-bins.tar.gz "https://go.kubebuilder.io/test-tools/${K8S_VERSION}/${GOOS}/${GOARCH}"
tar -xvzf envtest-bins.tar.gz
rm envtest-bins.tar.gz
