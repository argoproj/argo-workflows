# Recursion

Templates can recursively invoke each other! In this variation of the above coin-flip template, we continue to flip coins until it comes up heads.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: coinflip-recursive-
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
        template: heads                 # call heads template if "heads"
        when: "{{steps.flip-coin.outputs.result}} == heads"
      - name: tails                     # keep flipping coins if "tails"
        template: coinflip
        when: "{{steps.flip-coin.outputs.result}} == tails"

  - name: flip-coin
    script:
      image: python:alpine3.6
      command: [python]
      source: |
        import random
        result = "heads" if random.randint(0,1) == 0 else "tails"
        print(result)

  - name: heads
    container:
      image: alpine:3.6
      command: [sh, -c]
      args: ["echo \"it was heads\""]
```

Here's the result of a couple of runs of coin-flip for comparison.

```bash
argo get coinflip-recursive-tzcb5

STEP                         PODNAME                              MESSAGE
 ✔ coinflip-recursive-vhph5
 ├───✔ flip-coin             coinflip-recursive-vhph5-2123890397
 └─┬─✔ heads                 coinflip-recursive-vhph5-128690560
   └─○ tails

STEP                          PODNAME                              MESSAGE
 ✔ coinflip-recursive-tzcb5
 ├───✔ flip-coin              coinflip-recursive-tzcb5-322836820
 └─┬─○ heads
   └─✔ tails
     ├───✔ flip-coin          coinflip-recursive-tzcb5-1863890320
     └─┬─○ heads
       └─✔ tails
         ├───✔ flip-coin      coinflip-recursive-tzcb5-1768147140
         └─┬─○ heads
           └─✔ tails
             ├───✔ flip-coin  coinflip-recursive-tzcb5-4080411136
             └─┬─✔ heads      coinflip-recursive-tzcb5-4080323273
               └─○ tails
```

In the first run, the coin immediately comes up heads and we stop. In the second run, the coin comes up tail three times before it finally comes up heads and we stop.
