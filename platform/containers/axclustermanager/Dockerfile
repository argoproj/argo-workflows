FROM %%ARGO_BASE_REGISTRY%%/argobase/kubernetes-distro:ax-v1.6.7-6341-095136c-050831f

# Remove built-in kubeup code and copy AX version to container.
RUN rm -rf /kubernetes/cluster
ADD docker_build/cluster /kubernetes/cluster

ENV AX_KUBE_VERSION ax-v1.6.7-6341-095136c-050831f
ENV AX_AWS_IMAGE_NAME k8s-1.6-debian-jessie-amd64-hvm-ebs-2017-05-02

ENV AX_KUBERNETES_ROOT /kubernetes
ENV PYTHONPATH /ax/python
ENV USER root
ENV PATH /opt/google-cloud-sdk/bin:/kubernetes/platforms/linux/amd64:/ax/bin:${PATH}

USER root

ADD kubectl /opt/google-cloud-sdk/bin/kubectl

COPY docker_build/root /

# Add ARGO cluster management CLI
COPY argocluster.py /ax/bin/argocluster
RUN chmod a+x /ax/bin/argocluster

ADD package_bootstrap.sh /tmp/package_bootstrap.sh
RUN /tmp/package_bootstrap.sh

COPY motd /etc/motd
RUN echo '[ ! -z "$TERM" -a -r /etc/motd ] && cat /etc/motd' >> /etc/bash.bashrc
RUN echo 'PS1="argo cluster ops> "' >> /root/.bashrc

# some pip installs
# TODO: Need to move pip installs here so that it does not have dependency with kubernetes-distro container
RUN pip install anytree==2.2.1 prompt-toolkit==1.0.15

WORKDIR /ax/bin

CMD ["/bin/bash","-c","cat /etc/motd && python -m clustercli.main_cli"]
