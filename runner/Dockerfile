FROM summerwind/actions-runner-dind
USER runner
# install build tools
RUN sudo add-apt-repository ppa:longsleep/golang-backports -y \
  && sudo curl -fsSLo /usr/share/keyrings/kubernetes-archive-keyring.gpg https://packages.cloud.google.com/apt/doc/apt-key.gpg \
  && echo "deb [signed-by=/usr/share/keyrings/kubernetes-archive-keyring.gpg] https://apt.kubernetes.io/ kubernetes-xenial main" | sudo tee /etc/apt/sources.list.d/kubernetes.list
RUN sudo apt update -y \
  && sudo apt-get install kubectl golang-go python3 make git psmisc libssl-dev mysql-client ca-certificates cmake clang llvm pkg-config lcov coreutils autoconf automake autotools-dev libtool xutils-dev -y \
  && sudo rm -rf /var/lib/apt/lists/*
# install s3
RUN python3 -m pip install --upgrade pip \
  && pip3 install awscli
# install go
COPY --from=golang:1.15.12-alpine /usr/local/go/ /usr/local/go/
ENV PATH="/usr/local/go/bin:${PATH}"
# install k3d
RUN sudo curl -s https://raw.githubusercontent.com/rancher/k3d/main/install.sh | bash
# install
RUN sudo mkdir -p /tmp/infra && sudo mkdir -p /tmp/infra/bin
COPY --from=zhihanz/test-infra:master /infra /tmp/infra/bin/infra
COPY --from=zhihanz/test-infra:master /Makefile /tmp/infra/Makefile
COPY --from=zhihanz/test-infra:master /manifests /tmp/infra/manifests/