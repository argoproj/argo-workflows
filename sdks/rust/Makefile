# https://doc.rust-lang.org/cargo/reference/config.html?highlight=CARGO_REGISTRY_TOKEN#credentials
# https://github.com/OpenAPITools/openapi-generator/blob/master/.github/workflows/samples-rust.yaml
# https://openapi-generator.tech/docs/generators/rust/

GIT_TAG               := $(shell git describe --exact-match --tags --abbrev=0 2> /dev/null || echo untagged)

ifeq ($(GIT_TAG),untagged)
VERSION               := 0.0.0-latest
else
# remove the "v" prefix, not allowed
# https://stackoverflow.com/a/19571504
VERSION               := $(GIT_TAG:v%=%)
endif

# work dir
WD                    := $(shell echo "`pwd`/client")

DOCKER = docker run --rm -v $(WD):/wd --workdir /wd
CHOWN = chown -R $(shell id -u):$(shell id -g)

publish: generate
	cargo publish --token ${CARGO_REGISTRY_TOKEN}

generate:
	rm -Rf $(WD)
	mkdir -p $(WD)
	cat ../../api/openapi-spec/swagger.json | \
		sed 's/io.k8s.api.core.v1.//' | \
		sed 's/io.k8s.apimachinery.pkg.apis.meta.v1.//' \
		> $(WD)/swagger.json
	cp ../../LICENSE $(WD)/LICENSE
	$(DOCKER) openapitools/openapi-generator-cli:v5.2.1 \
		generate \
		-i /wd/swagger.json \
		-g rust \
		-o /wd \
		-p hideGenerationTimestamp=true \
		-p serializationLibrary=jsonb \
		--additional-properties packageVersion=${VERSION} \
		--additional-properties useSingleRequestParameter=true \
		--additional-properties supportMultipleResponses=true \
		--additional-properties packageName="argo_workflows" \
		--additional-properties projectName="argo-workflows" \
		--additional-properties hideGenerationTimestamp=true \
		--remove-operation-id-prefix \
		--model-name-prefix '' \
		--model-name-suffix '' \
		--artifact-id argo-rust-client \
 		--global-property packageName=argo_workflows \
		--generate-alias-as-model
	# https://vsupalov.com/docker-shared-permissions/#set-the-docker-user-when-running-your-container
	$(CHOWN) $(WD) || sudo $(CHOWN) $(WD)

install: 
	# Set up environment for testing and dev-dependency
	cp -r ./tests ./client
	cargo add tokio \
		--manifest-path ./client/Cargo.toml \
		--dev \
		--features "macros"

test:
	../../hack/access-token.sh init
	env ARGO_TOKEN="`../../hack/access-token.sh get`" cargo test --manifest-path ./client/Cargo.toml