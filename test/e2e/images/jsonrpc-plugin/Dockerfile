FROM python:3.8

USER 1234

COPY main.py .

ENV PYTHONUNBUFFERED=0

# https://stackoverflow.com/questions/29663459/python-app-does-not-print-anything-when-running-detached-in-docker
CMD [ "python", "-u", "./main.py" ]
