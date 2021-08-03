HUB ?= zhihanz
TAG ?= latest
INFRA_CMD ?=  ./bin/infra

# Cluster settings
PROVIDER ?= kind
CLUSTER_NAME ?= fusebench
CPU ?= 3300m
MEMORY ?= 3Gi
ENABLE_LB ?= true
NAMESPACE ?= default

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
CHATBOT_TAG ?= zhihanz/chatbot:latest
CHATBOT_WEBHOOK_TOKEN ?= Not public
CHATBOT_GITHUB_TOKEN ?= Not public

# Github Runner Settings
RUNNER_TOKEN ?= Not public

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
	docker buildx build . -f ./infra/Dockerfile  --platform linux/amd64 --allow network.host --builder host -t ${HUB}/test-infra:${TAG} --push
docker-bot: build-bot
	docker buildx build . -f ./chatbots/Dockerfile  --platform linux/amd64 --allow network.host --builder host -t ${HUB}/chatbot:${TAG} --push
docker-runner: docker-infra
	docker buildx build . -f ./runner/Dockerfile  --platform linux/amd64 --allow network.host --builder host -t ${HUB}/runner:${TAG} --push

deploy-bot: build-infra
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
deploy-runner:
	kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.4.0/cert-manager.yaml
	kubectl wait --for=condition=Ready -n cert-manager pods --all --timeout 600s
	kubectl apply -f https://github.com/actions-runner-controller/actions-runner-controller/releases/download/v0.18.2/actions-runner-controller.yaml
	kubectl delete secret generic controller-manager -n actions-runner-system --ignore-not-found
	kubectl create secret generic controller-manager \
		-n actions-runner-system \
		--from-literal=github_token=${RUNNER_TOKEN}
	kubectl wait --for=condition=Ready -n actions-runner-system pods --all --timeout 600s
	kubectl apply -f runner/runner.yaml
	kubectl wait --for=condition=Ready -n runner-system pods --all --timeout=600s
delete-runner:
	kubectl delete -f runner/runner.yaml
	kubectl delete -f https://github.com/actions-runner-controller/actions-runner-controller/releases/download/v0.18.2/actions-runner-controller.yaml
	kubectl delete -f https://github.com/jetstack/cert-manager/releases/download/v1.4.0/cert-manager.yaml
minikube_start:
	minikube start --cpus 10 --memory 16384 --disk-size='30000mb' --driver=kvm2
deploy_local: deploy-bot deploy-runner
port_forward:
	kubectl port-forward service/chatbot-service -n chatbot-system ${CHATBOT_PORT}:${CHATBOT_PORT}

deploy: cluster_create resource_apply run_perf run_compare
# GCP sometimes takes longer than 30 tries when trying to delete nodes
# if k8s resources are not already cleared
clean: delete-bot delete-runner

cluster_create: cluster_running
	@if [ ${ENABLE_LB} = "true" ]; then\
            echo "Install metallb load balancer";\
            kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.10.2/manifests/namespace.yaml;\
            kubectl apply -f https://raw.githubusercontent.com/metallb/metallb/v0.10.2/manifests/metallb.yaml;\
            kubectl apply -f ./manifests/lb_configs.yaml;\
            kubectl apply -f ./manifests/config.yaml;\
    fi
resource_apply: resource_apply_current resource_apply_ref
resource_apply_config:
	${INFRA_CMD} ${PROVIDER} resource apply  \
		-v CLUSTER_NAME:${CLUSTER_NAME} \
		-v NAMESPACE=${NAMESPACE}\
		-f manifests/config.yaml
resource_apply_current: resource_apply_config
	${INFRA_CMD} ${PROVIDER} resource apply  \
		-v CLUSTER_NAME:${CLUSTER_NAME} \
		-v CURRENT=${CURRENT} -v REF=${REFERENCE} \
		-v CPU=${CPU} -v MEMORY=${MEMORY} -v NAMESPACE=${NAMESPACE}\
		-f manifests/current

resource_apply_ref: resource_apply_config
	${INFRA_CMD} ${PROVIDER} resource apply  \
		-v CLUSTER_NAME:${CLUSTER_NAME} \
		-v CURRENT=${CURRENT} -v REF=${REFERENCE} \
		-v CPU=${CPU} -v MEMORY=${MEMORY}  -v NAMESPACE=${NAMESPACE}\
		-f manifests/ref

resource_delete:
	${INFRA_CMD} ${PROVIDER} resource delete  \
		-v CLUSTER_NAME:${CLUSTER_NAME} \
		-v CURRENT=${CURRENT} -v REF=${REFERENCE} \
		-v CPU=${CPU} -v MEMORY=${MEMORY}  -v NAMESPACE=${NAMESPACE}\
		-f manifests/current
	${INFRA_CMD} ${PROVIDER} resource delete  \
		-v CLUSTER_NAME:${CLUSTER_NAME} \
		-v CURRENT=${CURRENT} -v REF=${REFERENCE} \
		-v CPU=${CPU} -v MEMORY=${MEMORY}  -v NAMESPACE=${NAMESPACE}\
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
		-v CURRENT=${CURRENT} -v REF=${REFERENCE}  -v NAMESPACE=${NAMESPACE}\
		-v REGION=${REGION} -v BUCKET=${BUCKET} -v SECRET_ID=${AWS_ACCESS_KEY_ID} -v SECRET_KEY=${AWS_SECRET_ACCESS_KEY} \
		-v ENDPOINT=${ENDPOINT} -v ITERATION=${ITERATION} \
		-f manifests/perfs/perf_current_job.yaml
run_ref_perf:
	${INFRA_CMD} ${PROVIDER} resource apply  \
		-v CLUSTER_NAME:${CLUSTER_NAME} \
		-v LEFT=report/${PR_NUMBER}/${UUID}/current -v RIGHT=report/${PR_NUMBER}/${UUID}/ref \
		-v CPU=${CPU} -v MEMORY=${MEMORY} \
		-v CURRENT=${CURRENT} -v REF=${REFERENCE} -v NAMESPACE=${NAMESPACE}\
		-v REGION=${REGION} -v BUCKET=${BUCKET} -v SECRET_ID=${AWS_ACCESS_KEY_ID} -v SECRET_KEY=${AWS_SECRET_ACCESS_KEY} \
		-v ENDPOINT=${ENDPOINT} -v ITERATION=${ITERATION} \
		-f manifests/perfs/perf_ref_job.yaml
perf_clean:
	${INFRA_CMD} ${PROVIDER} resource delete  \
		-v CLUSTER_NAME:${CLUSTER_NAME} \
		-v LEFT=report/${PR_NUMBER}/${UUID}/${CURRENT} -v RIGHT=report/${PR_NUMBER}/${UUID}/${REFERENCE} \
		-v CPU=${CPU} -v MEMORY=${MEMORY} -v NAMESPACE=${NAMESPACE}\
		-v CURRENT=${CURRENT} -v REF=${REFERENCE} \
		-v REGION=${REGION} -v BUCKET=${BUCKET} -v SECRET_ID=${AWS_ACCESS_KEY_ID} -v SECRET_KEY=${AWS_SECRET_ACCESS_KEY} \
		-v ENDPOINT=${ENDPOINT} -v ITERATION=${ITERATION} \
		-f manifests/perfs
run_compare:
	${INFRA_CMD} ${PROVIDER} resource apply  \
		-v CLUSTER_NAME:${CLUSTER_NAME} \
		-v LEFT=report/${PR_NUMBER}/${UUID}/current/ -v RIGHT=report/${PR_NUMBER}/${UUID}/ref/ \
		-v PATH=report/${PR_NUMBER}/${UUID} -v NAMESPACE=${NAMESPACE}\
		-v REGION=${REGION} -v BUCKET=${BUCKET} -v SECRET_ID=${AWS_ACCESS_KEY_ID} -v SECRET_KEY=${AWS_SECRET_ACCESS_KEY} \
		-v ENDPOINT=${ENDPOINT} \
		-f manifests/compare
compare_clean:
	${INFRA_CMD} ${PROVIDER} resource delete  \
		-v CLUSTER_NAME:${CLUSTER_NAME} \
		-v LEFT=report/${PR_NUMBER}/${UUID}/${CURRENT} -v RIGHT=report/${PR_NUMBER}/${UUID}/${REFERENCE} \
		-v PATH=report/${PR_NUMBER}/${UUID} -v NAMESPACE=${NAMESPACE}\
		-v CURRENT=${CURRENT} -v REF=${REFERENCE} \
		-v REGION=${REGION} -v BUCKET=${BUCKET} -v SECRET_ID=${AWS_ACCESS_KEY_ID} -v SECRET_KEY=${AWS_SECRET_ACCESS_KEY} \
		-v ENDPOINT=${ENDPOINT} \
		-f manifests/compare
.PHONY: deploy
