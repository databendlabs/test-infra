FROM docker:stable as docker
FROM golang:1.15.12
LABEL maintainer="The Datafuselabs Authors"
COPY ./ /app
WORKDIR /app
RUN make build

RUN apt-get update
RUN apt-get install -y apt-transport-https ca-certificates curl make git
RUN curl -fsSLo /usr/share/keyrings/kubernetes-archive-keyring.gpg https://packages.cloud.google.com/apt/doc/apt-key.gpg
RUN echo "deb [signed-by=/usr/share/keyrings/kubernetes-archive-keyring.gpg] https://apt.kubernetes.io/ kubernetes-xenial main" | tee /etc/apt/sources.list.d/kubernetes.list
RUN apt-get update
RUN apt-get install -y kubectl

# Install Docker CLI.
COPY --from=docker /usr/local/bin/docker /usr/local/bin/docker
ENTRYPOINT ["/bin/sh", "-c"]
