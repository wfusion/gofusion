#!/usr/bin/env bash

set -ex

# project path
APP="gofusion"
WORKDIR=$(cd $(dirname "$0") || exit; pwd)
WORKDIR=${WORKDIR%"$APP"*}$APP
cd "${WORKDIR}" || exit
OUTPUT_DIR="${WORKDIR}"/assets
OUTPUT="${OUTPUT_DIR}"/unittest
mkdir -p "${OUTPUT}"
touch "${OUTPUT}"/coverage.out

# dep install
go install gotest.tools/gotestsum@latest
go install github.com/axw/gocov/gocov@latest
go install github.com/matm/gocov-html/cmd/gocov-html@latest
if ! type -p java; then
  if [ -f /etc/centos-release ] || [ -f /etc/system-release ]; then
      yum update -y
      yum install -y wget python3 glibc-langpack-en
  elif [ -f /etc/lsb-release ] || [ -f /etc/debian_version ]; then
      apt update
      apt install default-jdk -y
  fi
fi

if ! type -p allure; then
    ALLURE_VERSION=2.24.1
    wget https://repo1.maven.org/maven2/io/qameta/allure/allure-commandline/${ALLURE_VERSION}/allure-commandline-${ALLURE_VERSION}.zip
    unzip allure-commandline-${ALLURE_VERSION}.zip -d /opt/
    mv /opt/allure-${ALLURE_VERSION} /opt/allure
    rm allure-commandline-${ALLURE_VERSION}.zip
    echo "export PATH=$PATH:/opt/allure/bin" >> ~/.bashrc
    source ~/.bashrc
    if test -f ~/.zshrc; then
      echo "export PATH=$PATH:/opt/allure/bin" >> ~/.zshrc
    fi
fi
if ! type -p allure-combine; then
  pip install allure-combine
fi

# env
export LANG=en_US.UTF-8
export LC_ALL=en_US.UTF-8
export JAVA_TOOL_OPTIONS='-Dfile.encoding="UTF-8" -Dsun.jnu.encoding="UTF-8"'
GOBIN=$(go env GOPATH)/bin

# run test
COVER_PKG=$(find -type d -printf '%P\n' | egrep -v '^$|^.git/*|^test/*|^assets/*|^.idea/*|^common/fus/*|^common/infra/drivers/orm/opengauss/*|^common/infra/asynq/*|^common/infra/metrics/*|^common/infra/watermill/*|^common/infra/rotatelog/*|^common/utils/gomonkey/*|^common/utils/sqlparser/*' | awk '{print "github.com/wfusion/gofusion/" $0}' | sed ':a;N;$!ba;s/\n/,/g')
"${GOBIN}"/gotestsum --junitfile "${OUTPUT}"/junit.xml -- -timeout 30m -parallel 1 ./test/... -coverpkg="${COVER_PKG}" -coverprofile="${OUTPUT}"/coverage.out -covermode count || true

# export test report
echo "export complete.xml"
/opt/allure/bin/allure generate "${OUTPUT}" -o "${OUTPUT}"/allure --clean
while [ ! -d "${OUTPUT}"/allure ]; do
  sleep 1
done
allure-combine --remove-temp-files --ignore-utf8-errors "${OUTPUT}"/allure --dest "${OUTPUT}"
"${GOBIN}"/minify -r -o "${OUTPUT}"/unittest.html "${OUTPUT}"/complete.html || true

# export test coverage report
echo "export coverage.html"
"${GOBIN}"/gocov convert "${OUTPUT}"/coverage.out > "${OUTPUT}"/coverage.json
"${GOBIN}"/gocov-html < "${OUTPUT}"/coverage.json > "${OUTPUT}"/gocov.html
"${GOBIN}"/minify -r -o "${OUTPUT}"/coverage.html "${OUTPUT}"/gocov.html || true

echo "export coverage.svg"
"${GOBIN}"/go-cover-treemap -coverprofile "${OUTPUT}"/coverage.out -only-folders > "${OUTPUT}"/coverage.svg

# remove temps
rm "${OUTPUT}"/coverage.out "${OUTPUT}"/coverage.json "${OUTPUT}"/junit.xml "${OUTPUT}"/complete.html "${OUTPUT}"/gocov.html
rm -rf "${OUTPUT}"/allure
