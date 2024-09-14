import jsyaml from 'js-yaml';

export function parse<T>(value: string): T {
    if (value.startsWith('{')) {
        return JSON.parse(value);
    }
    return jsyaml.load(value) as T;
}

export function stringify<T>(value: T, type: string) {
    return type === 'yaml' ? jsyaml.dump(value, {noRefs: true}) : JSON.stringify(value, null, '  ');
}

/** Use JSON.stringify() to compare given values */
export function isEqual<T>(value1: T, value2: T) {
    return JSON.stringify(value1) === JSON.stringify(value2);
}
