// Presentation is declarative and does not grant permission or change saved values.
export interface FieldPredicate {
    path: string;
    operator: 'Equals' | 'Exists';
    value?: any;
}
export interface FormField {
    path: string;
    label?: string;
    description?: string;
    widget?: string;
    readOnly?: boolean;
    group?: string;
    optionSource?: string;
    visibleWhen?: FieldPredicate;
    enabledWhen?: FieldPredicate;
}
export interface LayoutNode {
    id: string;
    parent?: string;
    kind: 'group' | 'tab' | 'tree';
    label?: string;
    collapsed?: boolean;
}
export interface FormDefinition {
    labels?: {[path: string]: string};
    options?: {[key: string]: any};
    fields?: FormField[];
    layout?: LayoutNode[];
}
export const parts = (path: string) => {
    if (path === '') return [];
    if (!path.startsWith('/') || /~(?![01])/.test(path)) throw new Error('Form paths must be JSON pointers: ' + path);
    return path
        .substring(1)
        .split('/')
        .map(x => x.replace(/~1/g, '/').replace(/~0/g, '~'));
};
export function at(value: any, path: string): {value?: any; exists: boolean} {
    for (const key of parts(path)) {
        if (value === null || typeof value !== 'object' || !Object.prototype.hasOwnProperty.call(value, key)) return {exists: false};
        value = value[key];
    }
    return {value, exists: value !== undefined};
}
export function setAt(value: any, path: string, next: any): any {
    const keys = parts(path);
    const update = (old: any, index: number): any => {
        if (index === keys.length) return next;
        const key = keys[index];
        const copy = Array.isArray(old) ? old.slice() : {...(old || {})};
        const result = update(Object.prototype.hasOwnProperty.call(copy, key) ? copy[key] : undefined, index + 1);
        if (result === undefined) delete copy[key];
        else Object.defineProperty(copy, key, {value: result, enumerable: true, configurable: true, writable: true});
        return copy;
    };
    return update(value, 0);
}
export function schemaAt(root: any, path: string): any {
    let schema = root;
    const resolve = () => {
        const seen = new Set<string>();
        while (schema?.$ref?.startsWith('#/definitions/')) {
            const ref = schema.$ref;
            if (seen.has(ref)) throw new Error('Recursive form schema requires YAML editing.');
            seen.add(ref);
            schema = at(root, ref.substring(1)).value;
        }
    };
    for (const key of parts(path)) {
        resolve();
        schema = schema?.type === 'array' && (key === '*' || /^(0|[1-9]\d*)$/.test(key)) ? schema.items : schema?.properties?.[key];
    }
    resolve();
    return schema;
}
export const sameValue = (a: any, b: any): boolean => {
    if (a === b) return true;
    if (!a || !b || typeof a !== 'object' || typeof b !== 'object' || Array.isArray(a) !== Array.isArray(b)) return false;
    const keys = Object.keys(a);
    return keys.length === Object.keys(b).length && keys.every(k => Object.prototype.hasOwnProperty.call(b, k) && sameValue(a[k], b[k]));
};
export function fieldFor(form: FormDefinition, path: string): FormField | undefined {
    const actual = parts(path);
    return (form.fields || [])
        .filter(field => {
            const pattern = parts(field.path);
            return pattern.length === actual.length && pattern.every((key, i) => key === actual[i] || (key === '*' && /^(0|[1-9]\d*)$/.test(actual[i])));
        })
        .sort((a, b) => parts(a.path).filter(k => k === '*').length - parts(b.path).filter(k => k === '*').length)[0];
}
export function predicateMatches(predicate: FieldPredicate | undefined, values: any, field?: FormField, path = ''): boolean {
    if (!predicate) return true;
    const actual = parts(path);
    const captures = field ? parts(field.path).flatMap((k, i) => (k === '*' ? [actual[i]] : [])) : [];
    let n = 0;
    const target = predicate.path
        .split('/')
        .map(k => (k === '*' ? captures[n++] : k))
        .join('/');
    if (n > captures.length) return false;
    const value = at(values, target);
    return predicate.operator === 'Exists' ? value.exists : value.exists && sameValue(value.value, predicate.value);
}
export function effectiveOptions(form: FormDefinition): {[path: string]: any} {
    const sources = Object.fromEntries(Object.entries(form.options || {}).filter(([key]) => key.startsWith('/')));
    for (const field of form.fields || []) if (field.optionSource) sources[field.path] = form.options?.[field.optionSource];
    return sources;
}
export function validateForm(form: FormDefinition, schema: any): string[] {
    const errors: string[] = [];
    const nodes = new Map<string, LayoutNode>();
    const seen = new Set<string>();
    for (const node of form.layout || []) {
        if (!node.id || nodes.has(node.id)) errors.push('Layout IDs must be unique and nonempty.');
        if (!['group', 'tab', 'tree'].includes(node.kind)) errors.push('Unsupported layout kind: ' + node.kind);
        nodes.set(node.id, node);
    }
    for (const node of nodes.values()) {
        const ancestors = new Set<string>([node.id]);
        let parent = node.parent;
        while (parent) {
            if (!nodes.has(parent) || ancestors.has(parent)) {
                errors.push('Layout has a missing parent or cycle: ' + node.id);
                break;
            }
            ancestors.add(parent);
            parent = nodes.get(parent).parent;
        }
    }
    for (const field of form.fields || []) {
        try {
            const target = schemaAt(schema, field.path);
            if (seen.has(field.path)) errors.push('Duplicate field path: ' + field.path);
            seen.add(field.path);
            if (!target) errors.push('Field does not exist in the parameter schema: ' + field.path);
            if (field.group && !nodes.has(field.group)) errors.push('Field has an unknown layout group: ' + field.path);
            if (field.group && parts(field.path).some((_, i) => schemaAt(schema, '/' + field.path.substring(1).split('/').slice(0, i).join('/'))?.type === 'array'))
                errors.push('Group the whole array instead of individual array fields: ' + field.path);
            const allowed: {[key: string]: boolean} = {
                text: target?.type === 'string',
                number: ['integer', 'number'].includes(target?.type),
                checkbox: target?.type === 'boolean',
                select: !!target?.enum || !!field.optionSource || !!form.options?.[field.path],
                object: target?.type === 'object',
                array: target?.type === 'array',
                table: target?.type === 'array' && schemaAt(schema, field.path + '/*')?.type === 'object',
                yaml: true,
                secret: true
            };
            if (field.widget && !allowed[field.widget]) errors.push('Widget is incompatible with the field: ' + field.path);
            if (field.optionSource && !form.options?.[field.optionSource]) errors.push('Option source is not declared: ' + field.path);
            for (const predicate of [field.visibleWhen, field.enabledWhen])
                if (predicate) {
                    if (!['Equals', 'Exists'].includes(predicate.operator) || !schemaAt(schema, predicate.path)) errors.push('Invalid condition: ' + field.path);
                    if (parts(predicate.path).filter(k => k === '*').length > parts(field.path).filter(k => k === '*').length)
                        errors.push('Condition has unbound array identity: ' + field.path);
                }
        } catch (e) {
            errors.push(e.message);
        }
    }
    const grouped = (form.fields || []).filter(f => f.group);
    for (const field of grouped) if (grouped.some(other => other !== field && field.path.startsWith(other.path + '/'))) errors.push('Layout fields overlap: ' + field.path);
    for (const [path, source] of Object.entries(effectiveOptions(form))) {
        try {
            if (!schemaAt(schema, path)) errors.push('Invalid option field: ' + path);
            for (const dependency of source?.dependsOn || []) if (!schemaAt(schema, dependency)) errors.push('Invalid option dependency: ' + dependency);
        } catch (e) {
            errors.push(e.message);
        }
    }
    return errors;
}
