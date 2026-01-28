# Conditionals

We also support conditional execution. The syntax is implemented by [`govaluate`](https://github.com/Knetic/govaluate) which offers the support for complex syntax. See in the example:

/// tab | YAML

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: coinflip-
spec:
  entrypoint: coinflip
  templates:
  - name: coinflip
    steps:
    # flip a coin
    - - name: flip-coin
        template: flip-coin
    # evaluate the result in parallel
    - - name: heads
        template: heads                       # call heads template if "heads"
        when: "{{steps.flip-coin.outputs.result}} == heads"
      - name: tails
        template: tails                       # call tails template if "tails"
        when: "{{steps.flip-coin.outputs.result}} == tails"
    - - name: flip-again
        template: flip-coin
    - - name: complex-condition
        template: heads-tails-or-twice-tails
        # call heads template if first flip was "heads" and second was "tails" OR both were "tails"
        when: >-
            ( {{steps.flip-coin.outputs.result}} == heads &&
              {{steps.flip-again.outputs.result}} == tails
            ) ||
            ( {{steps.flip-coin.outputs.result}} == tails &&
              {{steps.flip-again.outputs.result}} == tails )
      - name: heads-regex
        template: heads                       # call heads template if ~ "hea"
        when: "{{steps.flip-again.outputs.result}} =~ hea"
      - name: tails-regex
        template: tails                       # call heads template if ~ "tai"
        when: "{{steps.flip-again.outputs.result}} =~ tai"

  # Return heads or tails based on a random number
  - name: flip-coin
    script:
      image: python:alpine3.23
      command: [python]
      source: |
        import random
        result = "heads" if random.randint(0,1) == 0 else "tails"
        print(result)

  - name: heads
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo \"it was heads\""]

  - name: tails
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo \"it was tails\""]

  - name: heads-tails-or-twice-tails
    container:
      image: alpine:3.23
      command: [sh, -c]
      args: ["echo \"it was heads the first flip and tails the second. Or it was two times tails.\""]
```

///

/// tab | Python

```python
from hera.workflows import Container, Step, Steps, Workflow, script


@script(image="python:alpine3.6")
def flip_coin():
    import random
    result = "heads" if random.randint(0,1) == 0 else "tails"
    print(result)

with Workflow(
    generate_name="coinflip-",
    entrypoint="coinflip",
) as w:
    heads = Container(
        name="heads",
        args=['echo "it was heads"'],
        command=["sh", "-c"],
        image="alpine:3.6",
    )
    tails = Container(
        name="tails",
        args=['echo "it was tails"'],
        command=["sh", "-c"],
        image="alpine:3.6",
    )
    heads_tails_or_twice_tails = Container(
        name="heads-tails-or-twice-tails",
        image="alpine:3.6",
        command=["sh", "-c"],
        args=[
            'echo "it was heads the first flip and tails the second. Or it was two times tails."'
        ],
    )
    with Steps(name="coinflip") as steps:
        flip_coin_step = flip_coin(name="flip-coin")

        with steps.parallel():
            Step(
                name="heads",
                template="heads",
                when=f"{flip_coin_step.result} == heads",
            )
            Step(
                name="tails",
                template="tails",
                when=f"{flip_coin_step.result} == tails",
            )

        flip_again = flip_coin(name="flip-again")

        with steps.parallel():
            heads_tails_or_twice_tails(
                name="complex-condition",
                when=f"( {flip_coin_step.result} == heads && {flip_again.result} == tails) || ( {flip_coin_step.result} == tails && {flip_again.result} == tails )",
            )
            heads(
                name="heads-regex",
                when=f"{flip_again.result} =~ hea",
            )
            tails(
                name="tails-regex",
                when=f"{flip_again.result} =~ tai",
            )
```

///

<!-- markdownlint-disable MD046 -- allow indentation within the admonition -->
!!! Warning "Nested Quotes"
    If the parameter value contains quotes, it may invalidate the `govaluate` expression.
    To handle parameters with quotes, embed an [`expr` expression](../variables.md#expression) in the conditional.
    For example:

    ```yaml
    when: "{{=inputs.parameters['may-contain-quotes'] == 'example'}}"
    ```
<!-- markdownlint-enable MD046 -->
