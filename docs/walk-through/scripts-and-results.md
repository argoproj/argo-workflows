# Scripts And Results

Often, you just want a template that executes a script specified as a here-script (also known as a `here document`) in the workflow spec. This example shows how to do that:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: scripts-bash-
spec:
  entrypoint: bash-script-example
  templates:
  - name: bash-script-example
    steps:
    - - name: generate
        template: gen-random-int-bash
    - - name: print
        template: print-message
        arguments:
          parameters:
          - name: message
            value: "{{steps.generate.outputs.result}}"  # The result of the here-script

  - name: gen-random-int-bash
    script:
      image: debian:9.4
      command: [bash]
      source: |                                         # Contents of the here-script
        cat /dev/urandom | od -N2 -An -i | awk -v f=1 -v r=100 '{printf "%i\n", f + r * $1 / 65536}'

  - name: gen-random-int-python
    script:
      image: python:alpine3.6
      command: [python]
      source: |
        import random
        i = random.randint(1, 100)
        print(i)

  - name: gen-random-int-javascript
    script:
      image: node:9.1-alpine
      command: [node]
      source: |
        var rand = Math.floor(Math.random() * 100);
        console.log(rand);

  - name: gen-random-int-java
    script:
      image: eclipse-temurin:22.0.2_9-jdk
      command: [java] # the Java interpreter requires files to end in `.java`
      extension: java # the file will now end in `.java`
      source: |
        import java.util.*;

        public class Main {
            public static void main(String[] args) {
                System.out.println((int)(Math.random()*100));
            }
        }

  - name: gen-random-scala
    script:
      image: virtuslab/scala-cli:1.5.0
      command: [scala-cli] # the scala-cli requires file to end in either `.scala` or `.sc`
      extension: sc # the file will now end in `.sc`
      source: |
        import scala.util.Random
        println(Random.between(0, 100))

  - name: print-message
    inputs:
      parameters:
      - name: message
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo result was: {{inputs.parameters.message}}"]
```

You can specify a script body with the  `source` field.
This creates a temporary file which is passed as the final parameter to `command`, which should be an interpreter.
You can set a file extension with `extension` field.
