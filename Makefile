HUB ?= zhihanz
TAG ?= latest
INFRA_CMD ?=  ./bin/infra

# Cluster settings
PROVIDER ?= kind
CLUSTER_NAME ?= fusebench
CPU ?= 3300m
MEMORY ?= 3Gi
ENABLE_LB ?= true

# branch info for compare
CURRENT ?= v0.4.33-nightly
REFERENCE ?= v0.4.33-nightly
PR_NUMBER ?= 12
UUID ?= 233
ITERATION ?= 3

# S3 report storage
AWS_DEFAULT_REGION ?= Not public
BUCKET ?= Not public
AWS_ACCESS_KEY_ID ?= Not public
AWS_SECRET_ACCESS_KEY ?= Not public
ENDPOINT ?= Not public
REGION ?= Not public

# Chatbot settings
CHATBOT_ADDRESS ?= 0.0.0.0
CHATBOT_PORT ?= 7070
CHATBOT_TAG ?= zhihanz/chatbot:debian
CHATBOT_WEBHOOK_TOKEN ?= Not public
CHATBOT_GITHUB_TOKEN ?= Not public
DELETE_CLUSTER_AFTER_RUN ?= true
build: build-infra

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
	docker buildx build . -f ./chatbots/Dockerfile  --platform linux/amd64 --allow network.host --builder host -t ${HUB}/chatbot:${TAG} --push
deploy-bot:
	${INFRA_CMD} ${PROVIDER} resource apply  \
    		-v CLUSTER_NAME:${CLUSTER_NAME} \
    		-v ADDRESS=${CHATBOT_ADDRESS} -v PORT=${CHATBOT_PORT} \
    		-v WEBHOOK_TOKEN=${CHATBOT_WEBHOOK_TOKEN} -v GITHUB_TOKEN=${CHATBOT_GITHUB_TOKEN} \
    		-v CHATBOT_TAG=${CHATBOT_TAG} \
    		-v REGION=${REGION} -v BUCKET=${BUCKET} -v ENDPOINT=${ENDPOINT} \
    		-f chatbots/deploy
delete-bot:
	${INFRA_CMD} ${PROVIDER} resource delete  \
    		-v CLUSTER_NAME:${CLUSTER_NAME} \
    		-v ADDRESS=${CHATBOT_ADDRESS} -v PORT=${CHATBOT_PORT} \
    		-v WEBHOOK_TOKEN=${CHATBOT_WEBHOOK_TOKEN} -v GITHUB_TOKEN=${CHATBOT_GITHUB_TOKEN} \
    		-v CHATBOT_TAG=${CHATBOT_TAG} \
    		-v REGION=${REGION} -v BUCKET=${BUCKET} -v ENDPOINT=${ENDPOINT} \
    		-f chatbots/deploy
deploy: cluster_create resource_apply run_perf run_compare
# GCP sometimes takes longer than 30 tries when trying to delete nodes
# if k8s resources are not already cleared
clean: cluster_delete

cluster_create: cluster_running
	@if [ ${ENABLE_LB} = "true" ]; then\
            echo "Install metallb load balancer";\
            kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.10.2/manifests/namespace.yaml;\
            kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.10.2/manifests/metallb.yaml;\
            kubectl apply -f ./manifests/lb_configs.yaml;\
            kubectl apply -f ./manifests/config.yaml;\
    fi
resource_apply: resource_apply_current resource_apply_ref
resource_apply_current:
	${INFRA_CMD} ${PROVIDER} resource apply  \
		-v CLUSTER_NAME:${CLUSTER_NAME} \
		-v CURRENT=${CURRENT} -v REF=${REFERENCE} \
		-v CPU=${CPU} -v MEMORY=${MEMORY} \
		-f manifests/current

resource_apply_ref:
	${INFRA_CMD} ${PROVIDER} resource apply  \
		-v CLUSTER_NAME:${CLUSTER_NAME} \
		-v CURRENT=${CURRENT} -v REF=${REFERENCE} \
		-v CPU=${CPU} -v MEMORY=${MEMORY} \
		-f manifests/ref

resource_delete:
	${INFRA_CMD} ${PROVIDER} resource delete  \
		-v CLUSTER_NAME:${CLUSTER_NAME} \
		-v CURRENT=${CURRENT} -v REF=${REFERENCE} \
		-v CPU=${CPU} -v MEMORY=${MEMORY} \
		-f manifests/current
	${INFRA_CMD} ${PROVIDER} resource delete  \
		-v CLUSTER_NAME:${CLUSTER_NAME} \
		-v CURRENT=${CURRENT} -v REF=${REFERENCE} \
		-v CPU=${CPU} -v MEMORY=${MEMORY} \
		-f manifests/ref
cluster_delete:
	@if [ ${DELETE_CLUSTER_AFTER_RUN} = "true" ]; then\
            	${INFRA_CMD} ${PROVIDER} cluster delete  \
            		-v CLUSTER_NAME:${CLUSTER_NAME} \
            		-v CURRENT=${CURRENT} -v REF=${REFERENCE} \
            		-f manifests/cluster-${PROVIDER}.yaml\
    fi

run_perf: run_current_perf run_ref_perf

run_current_perf:
	${INFRA_CMD} ${PROVIDER} resource apply  \
		-v CLUSTER_NAME:${CLUSTER_NAME} \
		-v LEFT=report/${PR_NUMBER}/${UUID}/current -v RIGHT=report/${PR_NUMBER}/${UUID}/ref\
		-v CPU=${CPU} -v MEMORY=${MEMORY} \
		-v CURRENT=${CURRENT} -v REF=${REFERENCE} \
		-v REGION=${REGION} -v BUCKET=${BUCKET} -v SECRET_ID=${AWS_ACCESS_KEY_ID} -v SECRET_KEY=${AWS_SECRET_ACCESS_KEY} \
		-v ENDPOINT=${ENDPOINT} -v ITERATION=${ITERATION} \
		-f manifests/perfs/perf_current_job.yaml
run_ref_perf:
	${INFRA_CMD} ${PROVIDER} resource apply  \
		-v CLUSTER_NAME:${CLUSTER_NAME} \
		-v LEFT=report/${PR_NUMBER}/${UUID}/current -v RIGHT=report/${PR_NUMBER}/${UUID}/ref \
		-v CPU=${CPU} -v MEMORY=${MEMORY} \
		-v CURRENT=${CURRENT} -v REF=${REFERENCE} \
		-v REGION=${REGION} -v BUCKET=${BUCKET} -v SECRET_ID=${AWS_ACCESS_KEY_ID} -v SECRET_KEY=${AWS_SECRET_ACCESS_KEY} \
		-v ENDPOINT=${ENDPOINT} -v ITERATION=${ITERATION} \
		-f manifests/perfs/perf_ref_job.yaml
perf_clean:
	${INFRA_CMD} ${PROVIDER} resource delete  \
		-v CLUSTER_NAME:${CLUSTER_NAME} \
		-v LEFT=report/${PR_NUMBER}/${UUID}/${CURRENT} -v RIGHT=report/${PR_NUMBER}/${UUID}/${REFERENCE} \
		-v CPU=${CPU} -v MEMORY=${MEMORY} \
		-v CURRENT=${CURRENT} -v REF=${REFERENCE} \
		-v REGION=${REGION} -v BUCKET=${BUCKET} -v SECRET_ID=${AWS_ACCESS_KEY_ID} -v SECRET_KEY=${AWS_SECRET_ACCESS_KEY} \
		-v ENDPOINT=${ENDPOINT} -v ITERATION=${ITERATION} \
		-f manifests/perfs
run_compare:
	${INFRA_CMD} ${PROVIDER} resource apply  \
		-v CLUSTER_NAME:${CLUSTER_NAME} \
		-v LEFT=report/${PR_NUMBER}/${UUID}/current/ -v RIGHT=report/${PR_NUMBER}/${UUID}/ref/ \
		-v PATH=report/${PR_NUMBER}/${UUID} \
		-v REGION=${REGION} -v BUCKET=${BUCKET} -v SECRET_ID=${AWS_ACCESS_KEY_ID} -v SECRET_KEY=${AWS_SECRET_ACCESS_KEY} \
		-v ENDPOINT=${ENDPOINT} \
		-f manifests/compare
compare_clean:
	${INFRA_CMD} ${PROVIDER} resource delete  \
		-v CLUSTER_NAME:${CLUSTER_NAME} \
		-v LEFT=report/${PR_NUMBER}/${UUID}/${CURRENT} -v RIGHT=report/${PR_NUMBER}/${UUID}/${REFERENCE} \
		-v PATH=report/${PR_NUMBER}/${UUID} \
		-v CURRENT=${CURRENT} -v REF=${REFERENCE} \
		-v REGION=${REGION} -v BUCKET=${BUCKET} -v SECRET_ID=${AWS_ACCESS_KEY_ID} -v SECRET_KEY=${AWS_SECRET_ACCESS_KEY} \
		-v ENDPOINT=${ENDPOINT} \
		-f manifests/compare
.PHONY: deploy
