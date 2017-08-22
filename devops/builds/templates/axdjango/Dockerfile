FROM %%ARGO_BASE_REGISTRY%%/argobase/django:1.9-python3

# Install Python dependencies
COPY requirements.txt /tmp/requirements.txt

RUN pip install --upgrade pip \
    && pip install -r /tmp/requirements.txt \
    && apt-get update && apt-get install -y vim curl git
