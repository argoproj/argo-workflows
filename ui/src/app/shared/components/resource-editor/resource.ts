import jsyaml = require('js-yaml');

export function parse<T>(value: string) {
    if (value.startsWith('{')) {
        return JSON.parse(value);
    }
    return jsyaml.load(value);
}

export function stringify<T>(value: T) {
    return JSON.stringify(value, null, '  ');
}
