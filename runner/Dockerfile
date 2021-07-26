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
# install
WORKDIR app
COPY --from=zhihanz/test-infra:master /infra /infra
COPY --from=zhihanz/test-infra:master /Makefile /Makefile
COPY --from=zhihanz/test-infra:master /manifests /manifests/