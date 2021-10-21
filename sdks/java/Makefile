GIT_TAG               := $(shell git describe --exact-match --tags --abbrev=0 2> /dev/null || echo untagged)
RELEASE_TAG           := $(shell [[ "$(GIT_TAG)" =~ ^v[0-9]+\.[0-9]+\.[0-9]+.*$$ ]] && echo "true" || "false")
# "SNAPSHOT" is "latest" for Java
VERSION               := 0.0.0-SNAPSHOT
ifeq ($(RELEASE_TAG),true)
VERSION               := $(GIT_TAG)
endif
# remove the "v" prefix, not allowed
VERSION               := $(VERSION:v=)

# work dir
WD                    := $(shell echo "`pwd`/wd")

wd:
	mkdir -p wd

wd/settings.xml: wd
	cp settings.xml wd/settings.xml

wd/pom.xml: wd/settings.xml
	cat ../../api/openapi-spec/swagger.json | \
		sed 's/io.k8s.api.core.v1.//' | \
		sed 's/io.k8s.apimachinery.pkg.apis.meta.v1.//' \
		> wd/swagger.json
	docker run --rm -v $(WD):/wd openapitools/openapi-generator-cli:v4.3.1 \
		generate \
		-i /wd/swagger.json \
		-g java \
		-o /wd \
		-p hideGenerationTimestamp=true \
		-p dateLibrary=joda \
		--api-package io.argoproj.workflow.apis \
		--invoker-package io.argoproj.workflow \
		--model-package io.argoproj.workflow.models \
		--skip-validate-spec \
		--group-id io.argoproj.workflow \
		--artifact-id argo-client-java \
		--artifact-version $(VERSION) \
		--import-mappings Time=org.joda.time.DateTime \
		--import-mappings Affinity=io.kubernetes.client.openapi.models.V1Affinity \
		--import-mappings ConfigMapKeySelector=io.kubernetes.client.openapi.models.V1ConfigMapKeySelector \
		--import-mappings Container=io.kubernetes.client.openapi.models.V1Container \
		--import-mappings ContainerPort=io.kubernetes.client.openapi.models.V1ContainerPort \
		--import-mappings EnvFromSource=io.kubernetes.client.openapi.models.V1EnvFromSource \
		--import-mappings EnvVar=io.kubernetes.client.openapi.models.V1EnvVar \
		--import-mappings HostAlias=io.kubernetes.client.openapi.models.V1HostAlias \
		--import-mappings Lifecycle=io.kubernetes.client.openapi.models.V1Lifecycle \
		--import-mappings ListMeta=io.kubernetes.client.openapi.models.V1ListMeta \
		--import-mappings LocalObjectReference=io.kubernetes.client.openapi.models.V1LocalObjectReference \
		--import-mappings ObjectMeta=io.kubernetes.client.openapi.models.V1ObjectMeta \
		--import-mappings ObjectReference=io.kubernetes.client.openapi.models.V1ObjectReference \
		--import-mappings PersistentVolumeClaim=io.kubernetes.client.openapi.models.V1PersistentVolumeClaim \
		--import-mappings PodDisruptionBudgetSpec=io.kubernetes.client.openapi.models.V1beta1PodDisruptionBudgetSpec \
		--import-mappings PodDNSConfig=io.kubernetes.client.openapi.models.V1PodDNSConfig \
		--import-mappings PodSecurityContext=io.kubernetes.client.openapi.models.V1PodSecurityContext \
		--import-mappings Probe=io.kubernetes.client.openapi.models.V1Probe \
		--import-mappings ResourceRequirements=io.kubernetes.client.openapi.models.V1ResourceRequirements \
		--import-mappings SecretKeySelector=io.kubernetes.client.openapi.models.V1SecretKeySelector \
		--import-mappings SecurityContext=io.kubernetes.client.openapi.models.V1SecurityContext \
		--import-mappings Toleration=io.kubernetes.client.openapi.models.V1Toleration \
		--import-mappings Volume=io.kubernetes.client.openapi.models.V1Volume \
		--import-mappings VolumeDevice=io.kubernetes.client.openapi.models.V1VolumeDevice \
		--import-mappings VolumeMount=io.kubernetes.client.openapi.models.V1VolumeMount \
		--generate-alias-as-model
	# add the io.kubernetes:java-client to the deps
	sed 's/<dependencies>/<dependencies><dependency><groupId>io.kubernetes<\/groupId><artifactId>client-java<\/artifactId><version>9.0.2<\/version><\/dependency>/g' wd/pom.xml > tmp && mv tmp wd/pom.xml

clean:
	rm -Rf wd

generate: wd/pom.xml

build: generate wd/settings.xml
	docker run -v ~/.m2:/root/.m2 -v $(WD):/wd -w /wd maven:3-openjdk-8 \
		mvn install -DskipTests -Dmaven.javadoc.skip -s settings.xml

publish: build
	# https://help.github.com/en/packages/using-github-packages-with-your-projects-ecosystem/configuring-apache-maven-for-use-with-github-packages
	docker run -v ~/.m2:/root/.m2 $(WD):/wd -w /wd maven:3-openjdk-8 \
		mvn deploy -DskipTests -Dmaven.javadoc.skip -DaltDeploymentRepository=github::default::https://maven.pkg.github.com/argoproj/argo-workflows -s settings.xml