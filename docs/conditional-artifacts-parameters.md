# Conditional Artifacts and Parameters

> v3.1 and after

The Conditional Artifacts and Parameters feature enables to assign the Step/DAG level artifacts or parameters based on
expression. This introduces a new field `fromExpression: ...` under Step/DAG level output artifact and `expression: ...`
under step/DAG level output parameter. Both use the
[expr](https://github.com/antonmedv/expr/blob/master/docs/Language-Definition.md) syntax.

## Conditional Artifacts

```yaml

- name: coinflip
  steps:
    - - name: flip-coin
        template: flip-coin
    - - name: heads
        template: heads
        when: "{{steps.flip-coin.outputs.result}} == heads"
      - name: tails
        template: tails
        when: "{{steps.flip-coin.outputs.result}} == tails"
  outputs:
    artifacts:
      - name: result
        fromExpression: "steps['flip-coin'].outputs.result == 'heads' ? steps.heads.outputs.artifacts.headsresult : steps.tails.outputs.artifacts.tailsresult"

```

* [Steps artifacts example](https://raw.githubusercontent.com/argoproj/argo-workflows/master/examples/conditional-artifacts.yaml)
* [DAG artifacts example](https://raw.githubusercontent.com/argoproj/argo-workflows/master/examples/dag-conditional-artifacts.yaml)

## Conditional Parameters

```yaml
    - name: coinflip
      steps:
        - - name: flip-coin
            template: flip-coin
        - - name: heads
            template: heads
            when: "{{steps.flip-coin.outputs.result}} == heads"
          - name: tails
            template: tails
            when: "{{steps.flip-coin.outputs.result}} == tails"
      outputs:
        parameters:
          - name: stepresult
            valueFrom:
              expression: "steps['flip-coin'].outputs.result == 'heads' ? steps.heads.outputs.result : steps.tails.outputs.result"
```

* [Steps parameter example](https://raw.githubusercontent.com/argoproj/argo-workflows/master/examples/conditional-parameters.yaml)
* [DAG parameter example](https://raw.githubusercontent.com/argoproj/argo-workflows/master/examples/dag-conditional-parameters.yaml)

## Built-In Functions

Convenient functions added to support more use cases:

1. `asInt`    - convert the string to integer (e.g: `asInt('1')`)
2. `asFloat`  - convert the string to Float (e.g: `asFloat('1.23')`)
3. `string`   - convert the int/float to string (e.g: `string(1)`)
4. `jsonpath` - Extract the element from JSON using JSON Path (
   e.g: `jsonpath('{"employee":{"name":"sonoo","salary":56000,"married":true}}", "$.employee.name" )` )
5. [Sprig](http://masterminds.github.io/sprig/) - Support all `sprig` functions

* [Advanced example: fibonacci Sequence](https://raw.githubusercontent.com/argoproj/argo-workflows/master/examples/fibonacci-seq-conditional-param.yaml)

!!! NOTE
    Expressions will decode the `-` as operator if template name has `-`, it will fail the expression. So here solution
    for template name which has `-` in its name. `step['one-two-three'].outputs.artifacts`
