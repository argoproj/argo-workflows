# Dockerfile for axapproval

FROM %%BUILDER_IMAGE_ID%%
ENV PATH $PATH:/ax/bin
ENV PYTHONPATH=/ax/src
COPY docker_build/src /ax/src
COPY axapproval.py /ax/bin/
CMD ["/ax/bin/axapproval.py"]

