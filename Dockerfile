# Build the handler and install helm and kubectl
FROM golang:1.16-buster as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY base/ base/
COPY cmd/ cmd/
COPY experiment/ experiment/
COPY lib/ lib/
COPY utils/ utils/
COPY .handler.yaml .handler.yaml
COPY main.go main.go

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o /bin/handler main.go

# Install kubectl
RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
RUN chmod 755 kubectl
RUN cp kubectl /bin

# Install Helm 3
RUN curl -fsSL -o helm-v3.5.0-linux-amd64.tar.gz https://get.helm.sh/helm-v3.5.0-linux-amd64.tar.gz
RUN tar -zxvf helm-v3.5.0-linux-amd64.tar.gz
RUN linux-amd64/helm version

# Install Kustomize v3
RUN curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash
RUN cp kustomize /bin
RUN GOBIN=/bin go get fortio.org/fortio

# Small linux image with useful shell commands
FROM debian:buster-slim
WORKDIR /
COPY --from=builder /bin/handler /bin/handler
COPY --from=builder /bin/kubectl /bin/kubectl
COPY --from=builder /bin/kustomize /bin/kustomize
COPY --from=builder /workspace/linux-amd64/helm /bin/helm
COPY --from=builder /bin/fortio /bin/fortio

# Install git
RUN apt-get update && apt-get install -y git
