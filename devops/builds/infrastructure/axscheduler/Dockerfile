# Debug Dockerfile for jobscheduler

FROM %%BUILDER_IMAGE_ID%%

ENV PATH $PATH:/ax/bin
ENV PYTHONPATH /ax/src
EXPOSE 9890

COPY jobscheduler.py /ax/bin/jobscheduler
COPY docker_build/src /ax/src
CMD ["jobscheduler"]
