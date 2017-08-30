# Dockerfile to perform SCM operations as part of a workflow (e.g. checkout, merge, push)

FROM %%BUILDER_IMAGE_ID%%

# Git requires a user to be set for some operations to run without errors (e.g. merge). Since we override this with
# the --author option, the values here do not matter
RUN apk --no-cache --update add mercurial \
    && git config --global user.email "user@auto" \
    && git config --global user.name "user"

ENV PATH $PATH:/ax/bin
ENV PYTHONPATH /ax/src
COPY axscm.py /ax/bin/axscm
COPY docker_build/src /ax/src
CMD ["axscm"]
