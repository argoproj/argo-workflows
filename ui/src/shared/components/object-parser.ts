import YAML from 'yaml';

export type Lang = 'json' | 'yaml';

// Default is YAML 1.2, but Kubernetes uses YAML 1.1, which leads to subtle bugs.
// See https://github.com/argoproj/argo-workflows/issues/12205#issuecomment-2111572189
const yamlVersion = '1.1';

export function parse<T>(value: string): T {
    if (value.startsWith('{')) {
        return JSON.parse(value);
    }
    return YAML.parse(value, {version: yamlVersion, strict: false}) as T;
}

export function stringify<T>(value: T, type: Lang) {
    return type === 'yaml' ? YAML.stringify(value, {aliasDuplicateObjects: false, version: yamlVersion, strict: false}) : JSON.stringify(value, null, '  ');
}
