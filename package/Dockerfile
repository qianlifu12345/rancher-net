FROM rancher/agent-base:v0.3.0
ENV CNI v0.3.0

ADD strongswan.tar.gz /
RUN mkdir -p /opt/cni/bin /etc/cni/net.d && \
    curl -sfSL https://github.com/containernetworking/cni/releases/download/${CNI}/cni-${CNI}.tgz | \
       tar xvzf - -C /tmp ./loopback && \
    curl -sfSL https://github.com/rancher/rancher-cni-bridge/releases/download/v0.1.2/rancher-cni-bridge-v0.1.2.tar.gz | \
    tar xvzf - -C /tmp \
    && mkdir -p /opt/cni/bin \
    && mv /tmp/rancher-cni-bridge /opt/cni/bin/rancher-bridge \
    && mv /tmp/loopback /opt/cni/bin/loopback && \
    curl -sfSL https://github.com/rancher/rancher-cni-ipam/releases/download/v0.1.0/rancher-cni-ipam.tar.gz | \
    tar xvzf - -C /opt/cni/bin

COPY start.sh start-vxlan.sh rancher-net /usr/bin/
COPY bob-host-local /opt/cni/bin/host-local
CMD ["start.sh"]
