HUB ?= datafusedev
TAG ?= latest
INFRA_CMD ?=  ./bin/infra
PROVIDER ?= kind
CLUSTER_NAME ?= fusebench
CPU ?= 3300m
MEMORY ?= 3Gi

build: build-infra build-bot

build-infra:
	mkdir -p bin
	go build -o ./bin/infra ./infra/...

build-bot:
	mkdir -p bin
	go build -o ./bin/bot ./chatbots/cmd/...

unit-test:
	go test ./chatbots/...

test: unit-test

lint:
	cargo fmt
	cargo clippy -- -D warnings

docker: docker-bot docker-infra

docker-infra: build-infra
	docker build --network host -f infra/Dockerfile -t ${HUB}/test-infra:${TAG} .
docker-bot: build-bot
	docker build --network host -f chatbots/Dockerfile -t ${HUB}/bot:${TAG} .



deploy: cluster_create resource_apply
# GCP sometimes takes longer than 30 tries when trying to delete nodes
# if k8s resources are not already cleared
clean: resource_delete cluster_delete

cluster_create:
	${INFRA_CMD} ${PROVIDER} cluster create  \
		-v CLUSTER_NAME:${CLUSTER_NAME} \
		-v CURRENT=${CURRENT} -v REF=${REF} \
		-f manifests/cluster-${PROVIDER}.yaml
resource_apply:
	${INFRA_CMD} ${PROVIDER} resource apply  \
		-v CLUSTER_NAME:${CLUSTER_NAME} \
		-v CURRENT=${CURRENT} -v REF=${REF} \
		-v CPU=${CPU} -v MEMORY=${MEMORY} \
		-f manifests/resources
resource_delete:
	${INFRA_CMD} ${PROVIDER} resource delete  \
		-v CLUSTER_NAME:${CLUSTER_NAME} \
		-v CURRENT=${CURRENT} -v REF=${REF} \
		-v CPU=${CPU} -v MEMORY=${MEMORY} \
		-f manifests/resources
cluster_delete:
	${INFRA_CMD} ${PROVIDER} cluster delete  \
		-v CLUSTER_NAME:${CLUSTER_NAME} \
		-v CURRENT=${CURRENT} -v REF=${REF} \
		-f manifests/cluster-${PROVIDER}.yaml
run_perf:
	docker run -it --mount src=`pwd`,target=/test_container,type=bind datafuselabs/perf-tool:latest \
		-- bash
.PHONY: docker build deploy lint
