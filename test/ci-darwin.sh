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
go install github.com/axw/gocov/gocov@v1.1.0
go install github.com/matm/gocov-html/cmd/gocov-html@v1.4.0
go install github.com/tdewolff/minify/v2/cmd/minify@v2.21.2
if ! type -p allure; then
  brew install allure
fi
if ! type -p allure-combine; then
  pip install allure-combine
fi

# run test
COVER_PKG=$(find . -type d | sed 's|^./||' | sed 's|^\.$||' | egrep -v '^$|^.git/*|^test/*|^assets/*|^.idea/*|^common/fus/*|^common/infra/drivers/orm/opengauss/*|^common/infra/drivers/orm/sqlite/*|^common/infra/asynq/*|^common/infra/metrics/*|^common/infra/watermill/*|^common/infra/rotatelog/*|^common/utils/gomonkey/*|^common/utils/sqlparser/*' | awk '{print "github.com/wfusion/gofusion/" $0}' | tr '\n' ',' | sed 's/,$//')
gotestsum --junitfile "${OUTPUT}"/junit.xml -- -timeout 30m -parallel 1 ./test/... -coverpkg="${COVER_PKG}" -coverprofile="${OUTPUT}"/coverage.out -covermode count || true

# export test report
echo "export complete.xml"
allure generate "${OUTPUT}" -o "${OUTPUT}"/allure --clean
while [ ! -d "${OUTPUT}"/allure ]; do
  sleep 1
done
allure-combine --remove-temp-files --ignore-utf8-errors "${OUTPUT}"/allure --dest "${OUTPUT}"
minify -r -o "${OUTPUT}"/unittest.html "${OUTPUT}"/complete.html || true

# export test coverage report
echo "export coverage.html"
gocov convert "${OUTPUT}"/coverage.out > "${OUTPUT}"/coverage.json
gocov-html < "${OUTPUT}"/coverage.json > "${OUTPUT}"/gocov.html
minify -r -o "${OUTPUT}"/coverage.html "${OUTPUT}"/gocov.html || true

echo "export coverage.svg"
go-cover-treemap -coverprofile "${OUTPUT}"/coverage.out -only-folders > "${OUTPUT}"/coverage.svg

# remove temps
rm "${OUTPUT}"/coverage.out "${OUTPUT}"/coverage.json "${OUTPUT}"/junit.xml "${OUTPUT}"/gocov.html "${OUTPUT}"/complete.html
rm -rf "${OUTPUT}"/allure