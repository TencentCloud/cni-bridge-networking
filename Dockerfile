FROM golang:1.10

ADD . /go/src/github.com/dbdd4us/cni-bridge-networking

WORKDIR /go/src/github.com/dbdd4us/cni-bridge-networking

RUN go build -v --ldflags '-linkmode external -extldflags "-static"' -o dist/cni-bridge main.go

RUN curl -L --retry 5 https://github.com/containernetworking/plugins/releases/download/v0.7.1/cni-plugins-amd64-v0.7.1.tgz | tar -xz -C dist/

FROM busybox

COPY --from=0 /go/src/github.com/dbdd4us/cni-bridge-networking/install-cni.sh /install-cni.sh

COPY --from=0 /go/src/github.com/dbdd4us/cni-bridge-networking/dist/cni-bridge /usr/bin/cni-bridge
COPY --from=0 /go/src/github.com/dbdd4us/cni-bridge-networking/dist/bridge /opt/cni/bin/bridge
COPY --from=0 /go/src/github.com/dbdd4us/cni-bridge-networking/dist/host-local /opt/cni/bin/host-local
COPY --from=0 /go/src/github.com/dbdd4us/cni-bridge-networking/dist/loopback /opt/cni/bin/loopback

