FROM 'node:6.3.1'

ADD ./src /app/src
ADD ./package.json /app/package.json
ADD ./yarn.lock /app/yarn.lock

WORKDIR /app
RUN npm install -g yarn && yarn install

ADD ./tsconfig.json /app/tsconfig.json
ADD ./tslint.json /app/tslint.json
ADD ./.gitignore /app/.gitignore

RUN npm run build

WORKDIR /tmp

RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.6.4/bin/linux/amd64/kubectl && chmod +x kubectl && mv ./kubectl /usr/bin/ && \
    curl -LO https://download.docker.com/linux/static/stable/x86_64/docker-17.03.0-ce.tgz && tar -xzf docker-17.03.0-ce.tgz && mv ./docker/docker /usr/bin/

EXPOSE 8080
