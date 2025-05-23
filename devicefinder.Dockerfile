ARG TARGETARCH=amd64

FROM --platform=linux/$TARGETARCH brew.registry.redhat.io/rh-osbs/openshift-golang-builder:v1.23 AS builder

WORKDIR /workspace
COPY . .

RUN make build-devicefinder

# Change this once ubi10 moves out of beta
FROM registry.redhat.io/ubi10-beta/ubi:latest

COPY --from=builder /workspace/_output/bin/devicefinder /usr/bin/

ENTRYPOINT ["/usr/bin/devicefinder"]

LABEL \
    com.redhat.component="Device Finder image for OpenShift Fusion Access Operator" \
    description="" \
    io.k8s.display-name="Device Finder image for OpenShift Fusion Access Operator" \
    io.k8s.description="" \
    io.openshift.tags="openshift,storage,scale" \
    distribution-scope="public" \
    name="openshift-fusion-access-devicefinder" \
    summary="Device Finder" \
    release="v1.0" \
    version="v1.0" \
    maintainer="Red Hat jgil@redhat.com" \
    url="https://github.com/openshift-storage-scale/openshift-fusion-access-operator.git" \
    vendor="Red Hat, Inc." \
    License="Apache License 2.0"
