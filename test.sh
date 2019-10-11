#!/usr/bin/env bash

# 在我的开发环境下额外做一些工作
if [[ $IS_CHEF_DEV_ENV == "true" ]]; then
    echo "CHEFERASEME run add_go_license..."
    add_go_license -d ./

    echo "CHEFERASEME run gofmt check..."
    gofiles=$(git diff --name-only --diff-filter=ACM | grep '.go$')
    if [ ! -z "$gofiles" ]; then
        #echo "CHEFERASEME mod gofiles exist:" $gofiles
        unformatted=$(gofmt -l $gofiles)
        if [ ! -z "$unformatted" ]; then
            echo "Go files should be formatted with gofmt. Please run:"
            for fn in $unformatted; do
                echo "  gofmt -w $PWD/$fn"
            done
            #exit 1
        else
            echo "Go files be formatted."
        fi
    else
        echo "CHEFERASEME mod gofiles not exist."
    fi
fi

# 跑 go test 生成测试覆盖率
echo "CHEFERASEME run coverage test..."
echo "" > coverage.txt

if [ ! -f "pkg/rtmp/testdata/test.flv" ]; then
    echo "CHEFERASEME test.flv not exist."
    if [ ! -d "pkg/rtmp/testdata" ]; then
        echo "CHEFERASEME mkdir."
        mkdir "pkg/rtmp/testdata"
    fi
    wget https://pengrl.com/images/other/source.200kbps.768x320.flv -O pkg/rtmp/testdata/test.flv
else
    echo "CHEFERASEME test.flv exist."
fi

for d in $(go list ./... | grep -v vendor | grep lal/pkg); do
    go test -race -coverprofile=profile.out -covermode=atomic $d
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm profile.out
    fi
done

# go test -race -coverprofile=profile.out -covermode=atomic && go tool cover -html=profile.out -o coverage.html && open coverage.html
# go test -test.bench=".*"
