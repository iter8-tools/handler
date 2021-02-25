# Build the handler and install helm and kubectl
FROM golang:1.15 as builder

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

# Small linux image with useful shell commands
FROM busybox:stable
WORKDIR /
COPY --from=builder /bin/handler /bin/handler
COPY --from=builder /bin/kubectl /bin/kubectl
COPY --from=builder /workspace/linux-amd64/helm /bin/helm
