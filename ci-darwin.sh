#!/usr/bin/env bash

set -ex

#go install gotest.tools/gotestsum@latest
#go get github.com/t-yuki/gocover-cobertura
#go mod tidy -v

COVER_PKG=$(find . -type d | sed 's|^./||' | sed 's|^\.$||' | egrep -v '^$|^.git/*|^test/*|^assets/*|^.idea/*|^common/fus*|^common/infra/drivers/orm/opengauss/*|^common/infra/asynq/*|^common/infra/metrics/*|^common/infra/watermill/*|^common/utils/gomonkey/*|^common/utils/sqlparser/*' | awk '{print "github.com/wfusion/gofusion/" $0}' | tr '\n' ',' | sed 's/,$//')

gotestsum --junitfile assets/junit.xml -- -p 1 -parallel 1 -timeout 30m ./test/... -coverpkg="${COVER_PKG}" -coverprofile=assets/coverage.out -covermode count || true

echo "export coverage.xml"
gocover-cobertura < assets/coverage.out > assets/coverage.xml

echo "export coverage.html"
go tool cover -func=assets/coverage.out
go tool cover -html=assets/coverage.out -o assets/coverage.html

echo "export coverage.svg"
go-cover-treemap -coverprofile assets/coverage.out -only-folders > assets/coverage.svg