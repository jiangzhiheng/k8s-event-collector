export  DATE:=${shell date "+%Y%m%d%H%M"}
REGISTRY ?= jiangzhiheng
IMAGE    ?= $(REGISTRY)/k8s-event-collector
VERSION  ?= v0.1-${DATE}
BIN_DIR=_output/bin


init:
	mkdir -p ${BIN_DIR}
bin: init
	CGO_ENABLED=0 go build -o=${BIN_DIR}/k8s-event-collector cmd/main.go

.PHONY: container
container:
	docker build --platform=linux/amd64 -t $(IMAGE):$(VERSION) .
	docker push $(IMAGE):$(VERSION)

clean:
	rm -rf _output/%