# Dockerfile for gateway

FROM %%BUILDER_IMAGE_ID%%

RUN apk --update add nginx

ENV AXHOME=/ax
ENV PYTHONPATH=$AXHOME/src
ENV DJANGOHOME $AXHOME/src/ax/devops/apps/gateway

RUN mkdir -p $AXHOME/db $AXHOME/log
RUN mkdir -p /run/nginx

COPY run.sh \
    gateway.conf \
    gateway.ini \
    $AXHOME/
RUN chmod +x $AXHOME/run.sh
RUN touch /run/nginx/nginx.pid
RUN touch /var/log/cron.log
RUN ln -s $AXHOME/gateway.conf /etc/nginx/conf.d/gateway.conf

EXPOSE 8889

COPY docker_build/src $AXHOME/src

RUN python3 $DJANGOHOME/manage.py check \
    && python3 $DJANGOHOME/manage.py makemigrations \
    && python3 $DJANGOHOME/manage.py migrate \
    && python3 $DJANGOHOME/manage.py crontab add

CMD $AXHOME/run.sh
