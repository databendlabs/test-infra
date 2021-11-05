HUB=datafuselabs
TAG=cargo
PLATFORM=linux/amd64
dockerx:
	docker buildx build . -f ./Dockerfile  --platform ${PLATFORM} --allow network.host --builder host -t ${HUB}/stateful-runner:${TAG} --push