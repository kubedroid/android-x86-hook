GOPATH=$(shell pwd)

docker:
	sudo docker build --rm=false . -t quay.io/quamotion/android-x86-hook:7.1-r2

run:
	sudo docker run --rm -it quay.io/quamotion/android-x86-hook:7.1-r2 /bin/bash

debug:
	sudo docker run --rm -it quay.io/quamotion/android-x86-hook:7.1-r2 /bin/bash

test:
	echo $(GOPATH)
	cd src/android-x86-hook && dep ensure -vendor-only
	cd src/android-x86-hook && go test
