all: build

build: clean
	go install -ldflags "-X main.Version=${version}" applatix.io/vol_plugin

clean:
	rm -f ${GOPATH}/bin/vol_plugin

release:
	aws --profile portal s3 cp ${GOPATH}/bin/vol_plugin s3://ax-public/ax_vol_plugin/bin/1.2.0/ --grants read=uri=http://acs.amazonaws.com/groups/global/AllUsers

.PHONY: all build clean
