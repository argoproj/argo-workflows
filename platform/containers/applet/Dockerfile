FROM %%BUILDER_IMAGE_ID%%

COPY docker_build/root /
COPY applet.py /ax/bin/applet
ENV PYTHONPATH /ax/python
ENV PATH /ax/bin:${PATH}

ENTRYPOINT ["applet"]
