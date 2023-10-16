# Conditional Artifacts and Parameters

> v3.1 and after

You can set Step/DAG level artifacts or parameters based on an [expression](variables.md#expression).
Use `fromExpression` under a Step/DAG level output artifact and `expression` under a Step/DAG level output parameter.

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

* [Steps artifacts example](https://raw.githubusercontent.com/argoproj/argo-workflows/main/examples/conditional-artifacts.yaml)
* [DAG artifacts example](https://raw.githubusercontent.com/argoproj/argo-workflows/main/examples/dag-conditional-artifacts.yaml)

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

* [Steps parameter example](https://raw.githubusercontent.com/argoproj/argo-workflows/main/examples/conditional-parameters.yaml)
* [DAG parameter example](https://raw.githubusercontent.com/argoproj/argo-workflows/main/examples/dag-conditional-parameters.yaml)
* [Advanced example: fibonacci Sequence](https://raw.githubusercontent.com/argoproj/argo-workflows/main/examples/fibonacci-seq-conditional-param.yaml)
