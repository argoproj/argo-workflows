import YAML from 'yaml';

export function parse<T>(value: string): T {
    if (value.startsWith('{')) {
        return JSON.parse(value);
    }
    return YAML.parse(value, {
        // Default is YAML 1.2, but Kubernetes uses YAML 1.1, which leads to subtle bugs.
        // See https://github.com/argoproj/argo-workflows/issues/12205#issuecomment-2111572189
        version: '1.1',
        strict: false
    }) as T;
}

export function stringify<T>(value: T, type: string) {
    return type === 'yaml' ? YAML.stringify(value, {aliasDuplicateObjects: false}) : JSON.stringify(value, null, '  ');
}
