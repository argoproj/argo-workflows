"""Native Kubernetes admission checks in the dedicated local experiment only.

Complements e2e.py: structural admission is not business-model admission.
No remote credentials, clouds or existing Argo installations are used.
"""
import copy
import json
import os
import subprocess
import uuid

NS = 'argo-parameterization-lab'
GROUP = 'parameters.workflows.example.io'
CONTEXT = os.environ.get('ARGO_PARAMETERIZATION_CONTEXT', 'k3d-argo-parameterization')
assert CONTEXT == 'k3d-argo-parameterization', 'dedicated local cluster only'
COMMAND = ['kubectl', '--context', CONTEXT, '--request-timeout=15s']
NAME = 'schema-' + uuid.uuid4().hex[:8]


def kubectl(args, body=None, expected=True):
    result = subprocess.run(COMMAND + args, input=json.dumps(body) if body is not None else None,
                            text=True, capture_output=True, timeout=25)
    assert (result.returncode == 0) == expected, result.stderr or result.stdout
    return result.stdout if expected else (result.stderr or result.stdout)


def resource(kind, spec, suffix=''):
    return {'apiVersion': GROUP + '/v1alpha1', 'kind': kind,
            'metadata': {'name': NAME + suffix, 'namespace': NS}, 'spec': spec}


def main():
    crds = json.loads(kubectl(['get', 'crd', '-l',
                             'app.kubernetes.io/part-of=argo-parameterization-experiment', '-o', 'json']))['items']
    assert len(crds) == 12
    for crd in crds:
        version = crd['spec']['versions'][0]
        assert 'status' in version['subresources']
        spec = version['schema']['openAPIV3Schema']['properties']['spec']
        assert spec['type'] == 'object' and not spec.get('x-kubernetes-preserve-unknown-fields')
    binding = resource('WorkflowParameterBinding', {
        'workflowTemplate': 'native', 'parameterTemplate': 'model',
        'mappings': [{'path': '/zones', 'parameter': 'zones', 'encoding': 'json'}]})
    kubectl(['create', '--dry-run=server', '--validate=strict', '-f', '-'], binding)
    for mutation, message in [
        ({'workflowTemplate': 123}, 'string'),
        ({'workflowTemplte': 'typo'}, 'unknown field'),
        ({'mappings': [{'path': '/zones', 'parameter': 'zones', 'encoding': 'unrecognized'}]}, 'Unsupported value'),
    ]:
        invalid = copy.deepcopy(binding)
        invalid['spec'].update(mutation)
        assert message in kubectl(['create', '--dry-run=server', '--validate=strict', '-f', '-'], invalid, False)
    invalid = copy.deepcopy(binding)
    del invalid['spec']['workflowTemplate']
    assert 'Required value' in kubectl(['create', '--dry-run=server', '-f', '-'], invalid, False)

    # Open business JSON is retained rather than pruned by the structural envelope.
    for index, value in enumerate([False, 0, None, ['a', 2, None], {'unlisted': {'nested': False}}, 'text']):
        document = resource('ParameterSet', {'template': 'model', 'values': value}, '-' + str(index))
        actual = json.loads(kubectl(['create', '--dry-run=server', '-f', '-', '-o', 'json'], document))
        assert 'values' in actual['spec'] and actual['spec']['values'] == value

    # Spec clients cannot forge observed state; status has its own RBAC boundary.
    document = resource('ParameterSet', {'template': 'model', 'values': {'x': 1}})
    document['status'] = {'observedGeneration': 999, 'digest': 'forged'}
    created = json.loads(kubectl(['create', '-f', '-', '-o', 'json'], document))
    assert not created.get('status')
    denied = kubectl(['auth', 'can-i', 'update', 'parametersets.' + GROUP,
                     '--subresource=status', '--as=system:serviceaccount:' + NS + ':argo-server', '-n', NS], expected=False)
    assert denied.strip() == 'no'

    # Immutable model snapshots must also protect nested open JSON.
    revision = resource('ParameterRevision', {
        'set': NAME, 'template': 'model', 'model': {
            'dialect': 'http://json-schema.org/draft-07/schema#', 'schema': {'type': 'object'}},
        'values': {'nested': {'value': 1}}, 'sources': {}, 'errors': None,
        'digest': 'synthetic', 'actor': 'local-test', 'savedAt': '2026-01-01T00:00:00Z'})
    stored = json.loads(kubectl(['create', '-f', '-', '-o', 'json'], revision))
    # A no-op update must be legal; otherwise every mutation would appear protected.
    kubectl(['replace', '--dry-run=server', '-f', '-'], stored)
    stored['metadata'].setdefault('labels', {})['test-note'] = 'metadata-only'
    kubectl(['replace', '--dry-run=server', '-f', '-'], stored)
    for patch in [{'spec': {'digest': 'changed'}}, {'spec': {'values': {'nested': {'value': 2}}}}]:
        error = kubectl(['patch', 'parameterrevision', NAME, '-n', NS, '--type=merge',
                         '--dry-run=server', '-p', json.dumps(patch)], expected=False)
        assert 'immutable' in error
    print('PASS typed CRD required/type/enum/unknown checks, arbitrary JSON, status isolation and immutable nested snapshots')


if __name__ == '__main__':
    main()
