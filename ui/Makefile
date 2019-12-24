dist/app: node_modules
	docker run --rm -w /wd -v `pwd`:/wd --entrypoint /usr/local/bin/yarn node:11.15.0 build
node_modules:
	docker run --rm -w /wd -v `pwd`:/wd --entrypoint /usr/local/bin/yarn node:11.15.0 install --frozen-lockfile --ignore-optional --non-interactive
