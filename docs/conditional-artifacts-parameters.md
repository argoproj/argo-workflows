# Conditional Artifacts and Parameters
> v3.1 and after

The Conditional Artifacts and Parameters feature enables to assign the Step/ DAG level artifacts or parameters based on expression. Introduced new field `fromExpression` under Step/DAG level output artifact and 'Expression' under  step/DAG level output  parameter.
Both fields will support [expr](https://github.com/antonmedv/expr/blob/master/docs/Language-Definition.md) format expression.

###Additional Custom functions
Few custom function added to support more use cases.
1. `asInt`    - convert the string to integer (e.g: asInt('1'))
2. `asFloat`  - convert the string to Float (e.g: asFloat('1.23'))
3. `string`   - convert the  int/float to string (e.g: string(1))
4. `jsonpath` - Extract the element from Json using jsonpath (e.g: jsonpath('{"employee":{"name":"sonoo","salary":56000,"married":true}}", "$.employee.name" ) )
5. [sprig](http://masterminds.github.io/sprig/) - Support all `sprig` functions

##Conditional Artifacts
```yaml

apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: conditional-artifacts-
spec:
  entrypoint: main
  volumes:
    - name: workdir
      emptyDir: {}
  templates:
    - name: main
      steps:
        - - name: coinflipmain
            template: coinflip
        - - name: printmessage
            template: printmessage
            arguments:
              artifacts:
                - name: message
                  from: "{{steps.coinflipmain.outputs.artifacts.stepresult}}"
    
    - name: coinflip
      steps:
        - - name: flip-coin
            template: flipcoin1
        - - name: heads
            template: heads
            when: "{{steps.flip-coin.outputs.result}} == heads"
          - name: tails
            template: tails
            when: "{{steps.flip-coin.outputs.result}} == tails"
      outputs:
        artifacts:
          - name: stepresult
            fromExpression: "steps['flip-coin'].outputs.result == 'heads'?steps.heads.outputs.artifacts.headsresult:steps.tails.outputs.artifacts.tailsresult"
    
    - name: flipcoin1
      script:
        image: python:alpine3.6
        command: [python]
        source: |
          import random
          result = "heads" if random.randint(0,1) == 0 else "tails"
          print(result)
          
    - name: heads
      script:
        image: python:alpine3.6
        command: [python]
        source: |
          file = open("result.txt", "w")
          file.write("it was heads")
          file.close()
        volumeMounts:
          - name: workdir
            mountPath: /mnt/vol
      outputs:
        artifacts:
          - name: headsresult
            path: /result.txt
    - name: tails
      script:
        image: python:alpine3.6
        command: [python]
        source: |
          file = open("result.txt", "w")
          file.write("it was tails")
          file.close()
        volumeMounts:
          - name: workdir
            mountPath: /mnt/vol
      outputs:
        artifacts:
          - name: tailsresult
            path: /result.txt
    - name: printmessage
      inputs:
        artifacts:
          - name: message
            path: /tmp/message
      container:
        image: argoproj/argosay:v1
        command: [sh, -c]
        args: ["cat /tmp/message"]
```
##Conditional Parameters

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: conditional-parameter-
spec:
  entrypoint: main
  volumes:
    - name: workdir
      emptyDir: {}
  templates:
    - name: main
      steps:
        - - name: coinflipmain
            template: coinflip
        - - name: printmessage
            template: printmessage
            arguments:
              parameters:
                - name: message
                  value: "{{steps.coinflipmain.outputs.parameters.stepresult}}"
    
    - name: coinflip
      steps:
        - - name: flipcoin
            template: flipcoin1
        - - name: heads
            template: heads
            when: "{{steps.flipcoin.outputs.result}} == heads"
          - name: tails
            template: tails
            when: "{{steps.flipcoin.outputs.result}} == tails"
      outputs:
        parameters:
          - name: stepresult
            valueFrom:
              expression: "steps.flipcoin.outputs.result == 'heads'? steps.heads.outputs.result : steps.tails.outputs.result"
    
    - name: flipcoin1
      script:
        image: python:alpine3.6
        command: [python]
        source: |
          import random
          result = "heads" if random.randint(0,1) == 0 else "tails"
          print(result)
          
    - name: heads
      container:
        image: argoproj/argosay:v1
        command: [sh, -c]
        args: [" echo heads"]

    - name: tails
      container:
        image: argoproj/argosay:v1
        command: [sh, -c]
        args: [" echo tails"]

    - name: printmessage
      inputs:
        parameters:
          - name: message
            valueFrom:
              Expression: "steps.flipcoin.outputs.result == 'heads'? steps.heads.outputs.result : steps.tails.outputs.result"
      container:
        image: argoproj/argosay:v1
        command: [sh, -c]
        args: ["echo {{inputs.parameters.message}}"]
```
Advanced example: [Fibonacci Sequence](../examples/fibonacci-seq-conditional-param.yaml)

Note: Expr will decode the `-` as operator if template name has `-`, it will fail the expression. So here workaround for template name which has `-` in its name. `step['one-two-three'].outputs.artifacts`