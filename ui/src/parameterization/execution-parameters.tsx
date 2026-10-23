import React, {useEffect, useRef, useState} from 'react';
import * as YAML from 'yaml';

import {uiUrl} from '../shared/base';
import {Workflow} from '../shared/models';
import {request} from './api';
import {NodeInputs} from './node-inputs';
import {PagedRevisionDiff} from './paged-revision-diff';

const group = 'parameters.workflows.example.io';
export function ExecutionParameters({workflow}: {workflow: Workflow}) {
    const revision = workflow.metadata.annotations?.[group + '/revision'];
    const [record, setRecord] = useState<any>();
    const [diff, setDiff] = useState<any>();
    const [error, setError] = useState('');
    const [comparing, setComparing] = useState(false);
    const generation = useRef(0);
    useEffect(() => {
        let active = true;
        generation.current++;
        setComparing(false);
        setRecord(undefined);
        setError('');
        setDiff(undefined);
        if (revision)
            request(workflow.metadata.namespace, 'revisions/' + revision)
                .then(data => {
                    if (active) setRecord(data);
                })
                .catch(e => {
                    if (active) setError(e.response?.body?.message || e.message);
                });
        return () => {
            active = false;
            generation.current++;
        };
    }, [revision, workflow.metadata.namespace, workflow.metadata.uid]);
    if (!revision) return null;
    return (
        <>
            <NodeInputs workflow={workflow} />
            <details className='white-box' style={{margin: 16}}>
                <summary>Parameter revision and execution inputs</summary>
                <p>
                    Revision: <code>{revision}</code>. This is the saved execution snapshot.
                </p>
                {error && <p role='alert'>{error}</p>}
                {record && (
                    <>
                        <pre>{YAML.stringify(record.spec.values)}</pre>
                        <details>
                            <summary>Value sources</summary>
                            <pre>{YAML.stringify(record.spec.sources)}</pre>
                        </details>
                        <button
                            className='argo-button'
                            disabled={comparing}
                            onClick={async () => {
                                const currentGeneration = generation.current;
                                setComparing(true);
                                setError('');
                                setDiff(undefined);
                                try {
                                    const current = await request(workflow.metadata.namespace, 'sets/' + record.spec.set);
                                    const comparison = await request(workflow.metadata.namespace, 'execution-diff', 'post', {
                                        source: {workflow: workflow.metadata.name, uid: workflow.metadata.uid},
                                        revision: current.spec.revision
                                    });
                                    if (currentGeneration === generation.current) setDiff(comparison);
                                } catch (e) {
                                    if (currentGeneration === generation.current) setError(e.response?.body?.message || e.message);
                                } finally {
                                    if (currentGeneration === generation.current) setComparing(false);
                                }
                            }}>
                            Compare execution with current draft and definitions
                        </button>
                        <a
                            style={{marginLeft: 16}}
                            href={uiUrl(
                                'parameters/' +
                                    encodeURIComponent(workflow.metadata.namespace) +
                                    '?sourceWorkflow=' +
                                    encodeURIComponent(workflow.metadata.name) +
                                    '&sourceUID=' +
                                    encodeURIComponent(workflow.metadata.uid)
                            )}>
                            Edit a parameter version
                        </a>
                        {diff && (
                            <>
                                {diff.notices.map((notice: string) => (
                                    <p role='note' key={notice}>
                                        {notice}
                                    </p>
                                ))}
                                <PagedRevisionDiff namespace={workflow.metadata.namespace} comparison={diff} execution={true} />
                            </>
                        )}
                    </>
                )}
            </details>
        </>
    );
}
