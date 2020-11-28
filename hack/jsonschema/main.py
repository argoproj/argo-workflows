#!/usr/bin/env python3

import os
import functools

import json


def relative_path(filepath):
    return os.path.join(os.path.dirname(__file__), filepath)

def remove_intxx_formats(path, v):
    if len(path) > 0 and path[-1] == 'format' and (v == 'int32' or v == 'int64'):
        return None
    return v


def parameter_string_or_number(path, v):
    if path == 'definitions/io.argoproj.workflow.v1alpha1.Parameter/properties/default/type'.split('/') \
       or path == 'definitions/io.argoproj.workflow.v1alpha1.Parameter/properties/value/type'.split('/'):
        return ['string', 'number']
    return v


def k8s_string_or_int(path, v):
    if path == 'definitions/io.k8s.apimachinery.pkg.util.intstr.IntOrString/type'.split('/'):
        return ['string', 'integer']
    return v


def k8s_string_or_int_no_format(path, v):
    if path == 'definitions/io.k8s.apimachinery.pkg.util.intstr.IntOrString/format'.split('/'):
        return None
    return v

# transforms return None to filter the kv out
# or a transformed value

transforms = [
    remove_intxx_formats,
    parameter_string_or_number,
    k8s_string_or_int,
    k8s_string_or_int_no_format
]


def apply_transforms(transforms, path, val):
    return functools.reduce(lambda x, y: y(path, x), reversed(transforms), val)


def walk_transform(path, val):
    if isinstance(val, dict):
        return dict([(new_k, new_v) for (new_k, new_v) in
                     ((k, walk_transform(path + [k], v))
                      for k, v in val.items())
                     if new_v is not None])
    return apply_transforms(transforms, path, val)


def main():
    with open(relative_path('../../api/openapi-spec/swagger.json')) as schema_fp:
        swagger = json.load(schema_fp)

    definitions = swagger['definitions']
    argo_types = [
        'CronWorkflow',
        'ClusterWorkflowTemplate',
        'Workflow',
        'WorkflowEventBinding',
        'WorkflowTemplate'
    ]

    for argo_type in argo_types:
        definition = definitions[f'io.argoproj.workflow.v1alpha1.{argo_type}']['properties']
        definition['apiVersion']['const'] = 'argoproj.io/v1alpha1'
        definition['kind']['const'] = argo_type
    
    schema = {
        '$id': 'http://workflows.argoproj.io/workflows.json', # don't really know what this should be
		'$schema': 'http://json-schema.org/schema#',
		'type':    'object',
		'oneOf': [{'$ref': f'#/definitions/io.argoproj.workflow.v1alpha1.{argo_type}'} for argo_type in argo_types],
		'definitions': definitions
    }

    fixed_schema = walk_transform([], schema)
    
    with open(relative_path('../../api/jsonschema/schema.json'), 'w') as schema_out:
        json.dump(fixed_schema, schema_out, indent=4)

if __name__ == '__main__':
    main()
