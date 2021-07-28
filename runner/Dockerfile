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
# install rust dependencies

RUN curl https://sh.rustup.rs -sSf | \
    sh -s -- --component rustfmt clippy miri stable -y && \
    $HOME/.cargo/bin/rustup toolchain update nightly --allow-downgrade
ENV PATH=$HOME/.cargo/bin:$PATH

# install sccache
RUN cargo install sccache
RUN sccache --show-stats
WORKDIR app
COPY --from=zhihanz/test-infra:master /infra /infra
COPY --from=zhihanz/test-infra:master /Makefile /Makefile
COPY --from=zhihanz/test-infra:master /manifests /manifests/