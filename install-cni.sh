#!/bin/sh

set -e -x;

if [ -w "/host/opt/cni/bin/" ]; then
    cp /opt/cni/bin/* /host/opt/cni/bin/;
    echo "Wrote CNI binaries to /host/opt/cni/bin/";
fi;

while :; do sleep 3600; done;