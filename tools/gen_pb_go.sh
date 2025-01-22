#!/bin/bash -eu
#
# Copyright 2023 Osprober Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script generates Go code for the config protobufs.

PROTOC_VERSION="21.5"
PROJECT="osprober"

GOPATH=$(go env GOPATH)

if [ -z "$GOPATH" ]; then
  echo "Go environment is not setup correctly. Please look at"
  echo "https://golang.org/doc/code.html to set up Go environment."
  exit 1
fi
echo GOPATH=${GOPATH}

if [ -z ${PROJECTROOT+x} ]; then
  # If PROJECTROOT is not set, try to determine it from script's location
  SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
  if [[ $SCRIPTDIR == *"$PROJECT/tools"* ]]; then
    PROJECTROOT="${SCRIPTDIR}/../.."
  else
    echo "PROJECTROOT is not set and we are not able to determine PROJECTROOT"
    echo "from script's path. PROJECTROOT should be set such that project files "
    echo " are located at $PROJECT relative to the PROJECTROOT."
    exit 1
  fi
fi
echo PROJECTROOT=${PROJECTROOT}

# Make sure protobuf compilation is set up correctly.
echo "Install protoc from the internet."
echo "============================================"
sleep 1

if [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then
  os="linux"
else
  echo "OS unsupported by this this build script. Please install protoc manually."
fi

TMPDIR=$(mktemp -d)
pushd $TMPDIR

arch=$(uname -m)

protoc_package="protoc-${PROTOC_VERSION}-${os}-${arch}.zip"
protoc_package_url="https://github.com/google/protobuf/releases/download/v${PROTOC_VERSION}/${protoc_package}"

echo -e "Downloading protobuf compiler from..\n${protoc_package_url}"
echo "======================================================================"
wget "${protoc_package_url}"
unzip "${protoc_package}"
export protoc_path=${PWD}/bin/protoc
popd

function cleanup {
  echo "Removing temporary directory used for protoc installation: ${TMPDIR}"
  rm  -rf "${TMPDIR}"
}
trap cleanup EXIT

# Get go plugin for protoc
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Install cue from Go
go install cuelang.org/go/cmd/cue@latest

echo "Generating Go code and CUE schema for protobufs.."
echo "======================================================================"
# Generate protobuf code from the root directory to ensure proper import paths.
cd $PROJECTROOT

# Create a temporary director to generate protobuf Go files.
mkdir -p ${TMPDIR}/github.com/jumpojoy
CLOUDPROBER_REPO=github.com/cloudprober/cloudprober
pushd $PROJECTROOT/$PROJECT
CLOUDPROBER_TAG=$(go list -f "{{ .Version }}" -m ${CLOUDPROBER_REPO})
popd

git clone --depth 1 --branch  ${CLOUDPROBER_TAG} http://${CLOUDPROBER_REPO} ${TMPDIR}/github.com/cloudprober/cloudprober

rsync -mr --exclude='.git' --include='*/' --include='*.proto' --include='*.cue' --exclude='*' $PROJECT $TMPDIR/github.com/jumpojoy

pushd $TMPDIR

MODULE=github.com/jumpojoy/osprober

echo "Generating CUE schema from protobufs.."
cue import proto -I . ${MODULE}/surfacers/formated_file/proto/config.proto --proto_enum json -f

# Generate Go code for proto
find ${MODULE} -type d | \
  while read -r dir
  do
    # Ignore directories with no proto files.
    ls ${dir}/*.proto > /dev/null 2>&1 || continue
    ${protoc_path} --go-grpc_out=. --go_out=. ${dir}/*.proto
  done

# Copy generated files back to their original location.
find ${MODULE} \( -name *.pb.go -o -name *proto_gen.cue \) | \
  while read -r pbgofile
  do
    dst=${PROJECTROOT}/${pbgofile/github.com\/jumpojoy\//}
    cp "$pbgofile" "$dst"
  done

popd

cleanup
