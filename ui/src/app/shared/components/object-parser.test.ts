import {exampleWorkflowTemplate} from '../examples';
import {parse, stringify} from './object-parser';

describe('parse', () => {
    it('handles a valid JSON string', () => {
        expect(parse('{}')).toEqual({});
        expect(parse('{"a": 1}')).toEqual({a: 1});
    });

    it('handles a malformed JSON string', () => {
        expect(() => parse('{1}')).toThrow();
    });

    it('handles a valid YAML string', () => {
        expect(parse('')).toEqual(null);
        expect(parse('a: 1')).toEqual({a: 1});
    });

    it('handles a malformed YAML string', () => {
        expect(() => parse('!foo')).toThrow();
    });

    it('parses a YAML string as YAML 1.1, not YAML 1.2', () => {
        expect(parse('foo: 0755')).toEqual({foo: 493});
    });
});

describe('stringify', () => {
    const testWorkflowTemplate = exampleWorkflowTemplate('test');
    testWorkflowTemplate.metadata.name = 'test-workflowtemplate';

    it('encodes to YAML', () => {
        // Can't use toMatchInlineSnapshot() until we upgrade to Jest 30: https://github.com/jestjs/jest/issues/14305
        expect(stringify(testWorkflowTemplate, 'yaml')).toEqual(`\
metadata:
  name: test-workflowtemplate
  namespace: test
  labels:
    example: "true"
spec:
  workflowMetadata:
    labels:
      example: "true"
  entrypoint: argosay
  arguments:
    parameters:
      - name: message
        value: hello argo
  templates:
    - name: argosay
      inputs:
        parameters:
          - name: message
            value: "{{workflow.parameters.message}}"
      container:
        name: main
        image: argoproj/argosay:v2
        command:
          - /argosay
        args:
          - echo
          - "{{inputs.parameters.message}}"
  ttlStrategy:
    secondsAfterCompletion: 300
  podGC:
    strategy: OnPodCompletion
`);
    });

    it('encodes to JSON', () => {
        expect(stringify(testWorkflowTemplate, 'json')).toEqual(`{
  "metadata": {
    "name": "test-workflowtemplate",
    "namespace": "test",
    "labels": {
      "example": "true"
    }
  },
  "spec": {
    "workflowMetadata": {
      "labels": {
        "example": "true"
      }
    },
    "entrypoint": "argosay",
    "arguments": {
      "parameters": [
        {
          "name": "message",
          "value": "hello argo"
        }
      ]
    },
    "templates": [
      {
        "name": "argosay",
        "inputs": {
          "parameters": [
            {
              "name": "message",
              "value": "{{workflow.parameters.message}}"
            }
          ]
        },
        "container": {
          "name": "main",
          "image": "argoproj/argosay:v2",
          "command": [
            "/argosay"
          ],
          "args": [
            "echo",
            "{{inputs.parameters.message}}"
          ]
        }
      }
    ],
    "ttlStrategy": {
      "secondsAfterCompletion": 300
    },
    "podGC": {
      "strategy": "OnPodCompletion"
    }
  }
}`);
    });
});
