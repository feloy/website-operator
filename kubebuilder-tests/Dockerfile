FROM golang:1.13

# Tests
RUN (cd /usr/local && wget https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-1.16.4-linux-amd64.tar.gz -O - | tar zx)

RUN go get github.com/onsi/ginkgo/ginkgo && go get github.com/onsi/gomega/...
