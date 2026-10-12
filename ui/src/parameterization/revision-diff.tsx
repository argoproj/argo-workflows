import * as React from 'react';
import {useState} from 'react';

interface Change {
    op: string;
    path: string;
    from?: string;
    itemIdentity?: string;
    before: any;
    after: any;
}
interface Comparison {
    layers?: {name: string; available: boolean; notice?: string; total?: number; changes: Change[]}[];
    left?: string;
    right?: string;
    changes?: Change[];
    modelChanges?: Change[];
    sourceChanges?: Change[];
    modelChanged?: boolean;
    identityRulesChanged?: boolean;
    sensitivePaths?: string[];
    totals?: {changes: number; modelChanges: number; sourceChanges: number};
}
const operations: {[key: string]: string} = {add: 'Added', remove: 'Removed', replace: 'Changed', type: 'Type changed', move: 'Moved'};
const inside = (path: string, parent: string) => path === parent || path.startsWith(parent + '/');
function valueText(value: any, change: Change, before: boolean, sensitive: string[]): string {
    if (change.op === 'move') return before ? change.from || 'Unknown position' : change.path;
    if (before ? change.op === 'add' : change.op === 'remove') return 'Not present';
    // Only versioned reference metadata is displayable at sensitive paths. Do not
    // make the diff renderer an accidental plaintext view for malformed imports.
    if (sensitive.some(p => p !== change.path && inside(p, change.path))) return 'Contains sensitive fields; values hidden';
    const root = sensitive.find(p => inside(change.path, p));
    if (root && !/^\/(secretRef\/(name|key|version|namespace|store|uid|resourceVersion)|latestSecretRef\/(configMap|key))$/.test(change.path.slice(root.length))) {
        if (value?.latestSecretRef) return 'Latest at submission: ' + JSON.stringify({configMap: value.latestSecretRef.configMap, key: value.latestSecretRef.key});
        if (!value?.secretRef) return 'Sensitive value hidden';
        const ref = value.secretRef;
        return 'Secret reference: ' + JSON.stringify({name: ref.name, key: ref.key, version: ref.version, namespace: ref.namespace});
    }
    if (value === undefined) return 'Not present';
    return JSON.stringify(value, null, 2);
}
export function RevisionDiff({comparison}: {comparison: Comparison}) {
    const [filter, setFilter] = useState('');
    const sections = [
        {name: 'Parameter values', available: true, notice: '', total: comparison.totals?.changes, changes: comparison.changes || [], sensitive: comparison.sensitivePaths || []},
        {name: 'Parameter model', available: true, notice: '', total: comparison.totals?.modelChanges, changes: comparison.modelChanges || [], sensitive: []},
        {name: 'Value sources', available: true, notice: '', total: comparison.totals?.sourceChanges, changes: comparison.sourceChanges || [], sensitive: []},
        ...(comparison.layers || []).map(layer => ({...layer, total: layer.total ?? layer.changes.length, sensitive: []}))
    ];
    return (
        <section aria-label='Revision diff' className='revision-diff'>
            <p>
                <code>{comparison.left}</code> → <code>{comparison.right}</code>
            </p>
            {comparison.modelChanged && <p role='note'>The parameter model changed. Review schema and default changes alongside the values.</p>}
            {comparison.identityRulesChanged && <p role='note'>Array identity rules changed; arrays are compared as whole values to avoid an ambiguous item match.</p>}
            <label>
                Filter paths or item identities
                <input aria-label='Filter revision changes' value={filter} onChange={e => setFilter(e.target.value)} />
            </label>
            {sections.map(section => {
                const selected = section.changes.filter(c => (c.path + ' ' + (c.itemIdentity || '')).toLowerCase().includes(filter.toLowerCase()));
                const groups: {[key: string]: Change[]} = {};
                selected.forEach(c => {
                    const group = c.path.split('/')[1] || '(root)';
                    (groups[group] ||= []).push(c);
                });
                return (
                    <details open={true} key={section.name}>
                        <summary>
                            {section.name} ({selected.length}/{section.changes.length})
                        </summary>
                        {!section.available && <p role='note'>{section.notice}</p>}
                        {section.available && !section.changes.length && <p>{section.total ? 'Changes are on another page.' : 'No changes.'}</p>}
                        {!!section.changes.length && !selected.length && <p>No matching changes.</p>}
                        {Object.keys(groups).map(group => (
                            <details open={true} key={group}>
                                <summary>{group.replace(/~1/g, '/').replace(/~0/g, '~')}</summary>
                                <div className='revision-diff-table'>
                                    <table>
                                        <thead>
                                            <tr>
                                                <th>Change / path</th>
                                                <th>Before</th>
                                                <th>After</th>
                                            </tr>
                                        </thead>
                                        <tbody>
                                            {groups[group].map((c, i) => (
                                                <tr key={c.path + ':' + c.op + ':' + i}>
                                                    <td>
                                                        <strong>{operations[c.op] || c.op}</strong>
                                                        <br />
                                                        <code>{c.path || '/'}</code>
                                                        {c.itemIdentity && (
                                                            <p>
                                                                Item: <code>{c.itemIdentity}</code>
                                                            </p>
                                                        )}
                                                        {c.from && (
                                                            <p>
                                                                From <code>{c.from}</code>
                                                            </p>
                                                        )}
                                                    </td>
                                                    <td>
                                                        <pre>{valueText(c.before, c, true, section.sensitive)}</pre>
                                                    </td>
                                                    <td>
                                                        <pre>{valueText(c.after, c, false, section.sensitive)}</pre>
                                                    </td>
                                                </tr>
                                            ))}
                                        </tbody>
                                    </table>
                                </div>
                            </details>
                        ))}
                    </details>
                );
            })}
        </section>
    );
}
