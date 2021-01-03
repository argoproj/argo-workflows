
local d = import 'doc-util/main.libsonnet';

{
    workflow+: {
        v1alpha1 +: {
            workflow+: {
                '#new'+: d.func.withArgs([
                    d.arg('generateName', d.T.string),
                ]),
                new(
                    generateName,
                )::
                    super.metadata.withGenerateName(generateName) + {spec: {}}
            },
        },
    },
}