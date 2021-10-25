#!/bin/bash

set +x

set -o errexit
set -o nounset
set -o pipefail


# Patch generated code.
# args:
#   $1: output directory
_patch() {
    local output_dir="$1"

    echo "--- Patching generated code..."

    # Fix imports
    find "${output_dir}/openapi_client/" -type f -name \*.py -exec sed -i 's/import openapi_client\./import argo.workflows.client./g' {} +
    find "${output_dir}/openapi_client/" -type f -name \*.py -exec sed -i 's/from openapi_client/from argo.workflows.client/g' {} +
    find "${output_dir}/openapi_client/" -type f -name \*.py -exec sed -i 's/getattr(openapi_client\.models/getattr(argo.workflows.client.models/g' {} +
    find "${output_dir}/openapi_client/" -type f -name \*_v1_container.py -exec sed -i 's/name\=None/name\=\'\''/g' {} +

	# Replace io.k8s models with python kubernetes.client.library
    find "$output_dir/openapi_client/" -type f \
        -exec sed -i "/import IoK8s/d" {} \; \
        -exec sed -i "s/IoK8sApiCore\|IoK8sApimachineryPkgApisMeta//g" {} \;

    # Import kubernetes to the relevant files
    # find "$output_dir/openapi_client/" ! -path '*/__pycache__/*' -type f | while read fname; do
    # {
    #     set +o pipefail

    #     ln=$(grep -P "V1(?!alpha)" ${fname} >/dev/null && grep -n "class" ${fname} | head -n1 | awk '{print $1}' FS=":" || true)
    #     if [ ! -z "${ln}" ]; then
    #         let ln--

    #         grep -P 'V1(?!alpha)[a-zA-Z]+' -oh ${fname} | sort -u | while read model; do
    #             sed -i "${ln}i from kubernetes.client.models import ${model}" ${fname}
    #             let ln++
    #         done

    #         sed -i 's/^class/\n&/1' ${fname}
    #     fi
    # }
    # done

    # TODO: yxue: why?
    # Import all kubernetes models
    # {
    #     models="$output_dir/openapi_client/models/__init__.py"
    #     echo -e '\nfrom kubernetes.client.models import *' >> ${models}
    # }

    # Add __version__ to the client package
     echo -e "\nfrom .__about__ import __version__" \
         >> "$output_dir/openapi_client/__init__.py"

    # Create a Python subpackage
    mkdir -p "${output_dir}/${PACKAGE_NAME//[.]/\/}/"
    cp -rl ${output_dir}/openapi_client/* "${output_dir}/${PACKAGE_NAME//[.]/\/}/"

    echo "--- Done."
}

# Cleanup generated code.
# args:
#   $1: output directory
_cleanup() {
    local output_dir="$1"

    echo "--- Sync source files with the argo/ directory."

    local source="$(realpath $output_dir/${PACKAGE_NAME%%.*})"
    # nest the generated code under argo/ directory
    rsync \
        --link-dest="$source" \
        --recursive \
        --remove-source-files \
        --verbose \
        "$source" "$output_dir/argo/"

    echo "--- Cleanup."
    {
        set +o nounset;
        [ ! -z ${CLEANUP_DIRS} ] && rm -r ${CLEANUP_DIRS[@]}
    }

    echo "--- Done."
}

# Generates client.
# args:
#   $1: output directory
#   $2: path to openaAPI specifiaction
#   $2: path to openaAPI config
# env:
#   [required] CLIENT_VERSION   : Client version. Will be used in the comment sections of the generated code
#   [required] PACKAGE_NAME     : Name of the client package.
#   [required] KUBERNETES_BRANCH: Kubernetes branch name to get the swagger spec from
argo::generate::generate_client() {
    : "${CLIENT_VERSION?Must set CLIENT_VERSION env var}"
    : "${KUBERNETES_BRANCH?Must set KUBERNETES_BRANCH env var}"
    : "${PACKAGE_NAME?Must set PACKAGE_NAME env var}"

    echo "--- Generating client."

    local output_dir="$1"
    local openapi_spec="$2"
    local openapi_config="$3"
    echo $(pwd)
    mkdir -m 755 -p $output_dir
	docker run --user ${UID} --rm -v $(pwd):/local openapitools/openapi-generator-cli:v4.3.1 generate \
		-g python \
		-c /local/${openapi_config} \
		-i /local/${openapi_spec} \
		-o /local/${output_dir} \
 		--global-property modelTests=false \
 		--global-property packageName=${PACKAGE_NAME} \
 		--global-property packageVersion=${CLIENT_VERSION}

    CLEANUP_DIRS=( "${output_dir}/test" "${output_dir}/openapi_client" )

    _patch   $output_dir
    _cleanup $output_dir

    echo "--- Done."
}

args=("$@")

if [ "$0" = "$BASH_SOURCE" ] ; then
    argo::generate::generate_client ${args[@]}
fi
