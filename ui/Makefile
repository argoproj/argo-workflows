YARN=$(shell which yarn)

dist/app: node_modules src
ifneq ($(YARN),)
	yarn build
else
	docker run --rm -w /wd -v `pwd`:/wd --entrypoint /usr/local/bin/yarn node:11.15.0 build
endif
	touch dist/app

node_modules: package.json yarn.lock
ifneq ($(YARN),)
	yarn install --frozen-lockfile --ignore-optional --non-interactive
else
	docker run --rm -w /wd -v `pwd`:/wd --entrypoint /usr/local/bin/yarn node:11.15.0 install --frozen-lockfile --ignore-optional --non-interactive
endif
