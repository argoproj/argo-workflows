import * as React from 'react';

export function DefinitionValidation({document}: {document: any}) {
    if (!document.metadata?.uid) return null;
    const status = document.status || {};
    const condition = (status.conditions || []).find((item: any) => item.type === 'DefinitionValid');
    const current = condition && status.observedGeneration === document.metadata.generation;
    const label = !condition
        ? 'Not observed'
        : !current
          ? 'Validation pending for this generation'
          : condition.status === 'True'
            ? 'Static checks passed'
            : condition.status === 'False'
              ? 'Definition invalid'
              : 'Binding context required';
    return (
        <section aria-label='Definition validation'>
            <h5>{label}</h5>
            <p>Static definition checks do not grant execution permission. Preview and submit validate the selected inputs and current caller access.</p>
            {!condition && <p>The optional definition status controller has not reported a result.</p>}
            {current && <p>{condition.message}</p>}
            {current && (status.errors || []).length > 0 && (
                <ul>
                    {status.errors.map((error: any, index: number) => (
                        <li key={index}>
                            {error.code} · {error.path}: {error.message}
                        </li>
                    ))}
                </ul>
            )}
            {current && (status.resolvedReferences || []).length > 0 && (
                <details>
                    <summary>Observed definition references ({status.resolvedReferences.length})</summary>
                    <table>
                        <thead>
                            <tr>
                                <th>Definition</th>
                                <th>UID</th>
                                <th>Content digest</th>
                            </tr>
                        </thead>
                        <tbody>
                            {status.resolvedReferences.map((ref: any) => (
                                <tr key={ref.kind + '/' + ref.name}>
                                    <td>
                                        {ref.kind}/{ref.name}
                                    </td>
                                    <td>{ref.uid}</td>
                                    <td>
                                        <code>{ref.digest?.slice(0, 16)}</code>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </details>
            )}
        </section>
    );
}
