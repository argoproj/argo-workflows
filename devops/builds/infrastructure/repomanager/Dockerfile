# Dockerfile for repomanager

#FROM %%ARGO_BASE_REGISTRY%%/argobase/alpine:3.5
#
#RUN apk --no-cache --update add git
#
#ENV PATH $PATH:/ax/bin
#COPY docker_build/dist/repomanager /ax/bin/repomanager
#CMD ["repomanager"]

FROM %%BUILDER_IMAGE_ID%%

ENV PYTHONPATH=/ax/src
COPY docker_build/src /ax/src

COPY repomanager.py /ax/bin/repomanager
COPY run.sh /ax/bin/run.sh
RUN chmod +x /ax/bin/run.sh

CMD /ax/bin/run.sh
