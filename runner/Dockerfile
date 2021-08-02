FROM summerwind/actions-runner-dind
# install build tools
RUN sudo apt update -y \
  && sudo apt-get install python3 make git -y \
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
RUN sudo mkdir -p /tmp/infra && mkdir -p /tmp/infra/bin
COPY --from=zhihanz/test-infra:master /infra /tmp/infra/bin/infra
COPY --from=zhihanz/test-infra:master /Makefile /tmp/infra/Makefile
COPY --from=zhihanz/test-infra:master /manifests /tmp/infra/manifests/