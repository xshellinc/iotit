#!/usr/bin/env bash

configure_aws_cli(){
  aws --version
  aws configure set default.region us-west-2
  aws configure set default.output json
}

deploy() {
    if [ ! -z "$CIRCLE_TAG" ]; then

	    VERSION=$(git describe --tags)
        echo "Current tag is $VERSION"

        # Uploading to folder by tag for user-app
        aws s3 cp build/${VERSION}/. s3://iotit/${VERSION}/darwin/ --recursive --exclude "*" --include "iotit_${VERSION}_darwin_*" --exclude "*/*"
        aws s3 cp build/${VERSION}/. s3://iotit/${VERSION}/linux/ --recursive --exclude "*" --include "iotit_${VERSION}_linux_*" --exclude "*/*"
        aws s3 cp build/${VERSION}/. s3://iotit/${VERSION}/windows/ --recursive --exclude "*" --include "iotit_${VERSION}_windows_*" --exclude "*/*"

        curl http://iotit.s3-website-us-west-2.amazonaws.com/version.json | jq \
        --arg md5_linux_x86 "$(md5sum build/${VERSION}/iotit_${VERSION}_linux_386.tar.gz | cut -d ' ' -f1)" \
        --arg md5_linux_amd64 "$(md5sum build/${VERSION}/iotit_${VERSION}_linux_amd64.tar.gz | cut -d ' ' -f1)" \
        --arg md5_linux_arm "$(md5sum build/${VERSION}/iotit_${VERSION}_linux_arm.tar.gz | cut -d ' ' -f1)" \
        --arg md5_darwin_x86 "$(md5sum build/${VERSION}/iotit_${VERSION}_darwin_386.zip | cut -d ' ' -f1)" \
        --arg md5_darwin_amd64 "$(md5sum build/${VERSION}/iotit_${VERSION}_darwin_amd64.zip | cut -d ' ' -f1)" \
        --arg md5_windows_x86 "$(md5sum build/${VERSION}/iotit_${VERSION}_windows_386.zip | cut -d ' ' -f1)" \
        --arg md5_windows_amd64 "$(md5sum build/${VERSION}/iotit_${VERSION}_windows_amd64.zip | cut -d ' ' -f1)" \
        --arg sha1_linux_x86 "$(sha1sum build/${VERSION}/iotit_${VERSION}_linux_386.tar.gz | cut -d ' ' -f1)" \
        --arg sha1_linux_amd64 "$(sha1sum build/${VERSION}/iotit_${VERSION}_linux_amd64.tar.gz | cut -d ' ' -f1)" \
        --arg sha1_linux_arm "$(sha1sum build/${VERSION}/iotit_${VERSION}_linux_arm.tar.gz | cut -d ' ' -f1)" \
        --arg sha1_darwin_x86 "$(sha1sum build/${VERSION}/iotit_${VERSION}_darwin_386.zip | cut -d ' ' -f1)" \
        --arg sha1_darwin_amd64 "$(sha1sum build/${VERSION}/iotit_${VERSION}_darwin_amd64.zip | cut -d ' ' -f1)" \
        --arg sha1_windows_x86 "$(sha1sum build/${VERSION}/iotit_${VERSION}_windows_386.zip | cut -d ' ' -f1)" \
        --arg sha1_windows_amd64 "$(sha1sum build/${VERSION}/iotit_${VERSION}_windows_amd64.zip | cut -d ' ' -f1)" \
        --arg version_stable "$VERSION" \
        '.stable.md5sums.linux.x86 = $md5_linux_x86 | .stable.md5sums.linux.amd64 = $md5_linux_amd64 | .stable.md5sums.linux.arm = $md5_linux_arm | 
        .stable.md5sums.darwin.x86 = $md5_darwin_x86 | .stable.md5sums.darwin.amd64 = $md5_darwin_amd64 |
        .stable.md5sums.windows.x86 = $md5_windows_x86 | .stable.md5sums.windows.amd64 = $md5_windows_amd64 | 
        .stable.sha1sums.linux.x86 = $sha1_linux_x86 | .stable.sha1sums.linux.amd64 = $sha1_linux_amd64 | .stable.sha1sums.linux.arm = $sha1_linux_arm | 
        .stable.sha1sums.darwin.x86 = $sha1_darwin_x86 | .stable.sha1sums.darwin.amd64 = $sha1_darwin_amd64 |
        .stable.sha1sums.windows.x86 = $sha1_windows_x86 | .stable.sha1sums.windows.amd64 = $sha1_windows_amd64 |
        .stable.version = $version_stable' \
        > version.json
        aws s3 cp version.json s3://iotit/version.json
    elif [ "$CIRCLE_BRANCH" == "develop" ]; then

	    VERSION=$(git describe --tags)
	    echo "Current version is $VERSION"

	    # Uploading to latest folder for old versios support
        aws s3 rm  s3://iotit/latest --recursive
        aws s3 cp build/${VERSION}/. s3://iotit/latest/darwin/ --recursive --exclude "*" --include "iotit_${VERSION}_darwin_*" --exclude "*/*"
        aws s3 cp build/${VERSION}/. s3://iotit/latest/linux/ --recursive --exclude "*" --include "iotit_${VERSION}_linux_*" --exclude "*/*"
        aws s3 cp build/${VERSION}/. s3://iotit/latest/windows/ --recursive --exclude "*" --include "iotit_${VERSION}_windows_*" --exclude "*/*"

        curl http://iotit.s3-website-us-west-2.amazonaws.com/version.json | jq \
        --arg md5_linux_x86 "$(md5sum build/${VERSION}/iotit_${VERSION}_linux_386.tar.gz | cut -d ' ' -f1)" \
        --arg md5_linux_amd64 "$(md5sum build/${VERSION}/iotit_${VERSION}_linux_amd64.tar.gz | cut -d ' ' -f1)" \
        --arg md5_linux_arm "$(md5sum build/${VERSION}/iotit_${VERSION}_linux_arm.tar.gz | cut -d ' ' -f1)" \
        --arg md5_darwin_x86 "$(md5sum build/${VERSION}/iotit_${VERSION}_darwin_386.zip | cut -d ' ' -f1)" \
        --arg md5_darwin_amd64 "$(md5sum build/${VERSION}/iotit_${VERSION}_darwin_amd64.zip | cut -d ' ' -f1)" \
        --arg md5_windows_x86 "$(md5sum build/${VERSION}/iotit_${VERSION}_windows_386.zip | cut -d ' ' -f1)" \
        --arg md5_windows_amd64 "$(md5sum build/${VERSION}/iotit_${VERSION}_windows_amd64.zip | cut -d ' ' -f1)" \
        --arg sha1_linux_x86 "$(sha1sum build/${VERSION}/iotit_${VERSION}_linux_386.tar.gz | cut -d ' ' -f1)" \
        --arg sha1_linux_amd64 "$(sha1sum build/${VERSION}/iotit_${VERSION}_linux_amd64.tar.gz | cut -d ' ' -f1)" \
        --arg sha1_linux_arm "$(sha1sum build/${VERSION}/iotit_${VERSION}_linux_arm.tar.gz | cut -d ' ' -f1)" \
        --arg sha1_darwin_x86 "$(sha1sum build/${VERSION}/iotit_${VERSION}_darwin_386.zip | cut -d ' ' -f1)" \
        --arg sha1_darwin_amd64 "$(sha1sum build/${VERSION}/iotit_${VERSION}_darwin_amd64.zip | cut -d ' ' -f1)" \
        --arg sha1_windows_x86 "$(sha1sum build/${VERSION}/iotit_${VERSION}_windows_386.zip | cut -d ' ' -f1)" \
        --arg sha1_windows_amd64 "$(sha1sum build/${VERSION}/iotit_${VERSION}_windows_amd64.zip | cut -d ' ' -f1)" \
        --arg version_latest "$VERSION" \
        '.latest.md5sums.linux.x86 = $md5_linux_x86 | .latest.md5sums.linux.amd64 = $md5_linux_amd64 | .latest.md5sums.linux.arm = $md5_linux_arm | 
        .latest.md5sums.darwin.x86 = $md5_darwin_x86 | .latest.md5sums.darwin.amd64 = $md5_darwin_amd64 |
        .latest.md5sums.windows.x86 = $md5_windows_x86 | .latest.md5sums.windows.amd64 = $md5_windows_amd64 | 
        .latest.sha1sums.linux.x86 = $sha1_linux_x86 | .latest.sha1sums.linux.amd64 = $sha1_linux_amd64 | .latest.sha1sums.linux.arm = $sha1_linux_arm | 
        .latest.sha1sums.darwin.x86 = $sha1_darwin_x86 | .latest.sha1sums.darwin.amd64 = $sha1_darwin_amd64 |
        .latest.sha1sums.windows.x86 = $sha1_windows_x86 | .latest.sha1sums.windows.amd64 = $sha1_windows_amd64 |
        .latest.version = $version_latest' \
        > version.json
        aws s3 cp version.json s3://iotit/version.json
    else
	    echo 'Release was not tagged, artifact upload cancelled'
    fi
}

configure_aws_cli
deploy