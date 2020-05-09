FROM alpine:3.8
ENV KUSTOMIZE_VER 3.5.4

RUN apk --no-cache add curl gettext

RUN mkdir /working
WORKDIR /working

RUN curl -L https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv3.5.4/kustomize_v3.5.4_linux_amd64.tar.gz --output - | tar zx \
    && mv kustomize /usr/bin \
    && chmod +x /usr/bin/kustomize

ENTRYPOINT ["/usr/bin/kustomize"]
