import * as React from 'react';
import {useEffect, useState} from 'react';
import {RouteComponentProps} from 'react-router';

import {NodePhase} from '../../models';
import {uiUrl} from '../shared/base';
import {historyUrl} from '../shared/history';
import {RetryWatch} from '../shared/retry-watch';
import {services} from '../shared/services';

import './workflow-status-badge.scss';

export function WorkflowStatusBadge({history, match}: RouteComponentProps<any>) {
    const queryParams = new URLSearchParams(location.search);
    const namespace = match.params.namespace;
    const name = queryParams.get('name');
    const label = queryParams.get('label');
    const target = queryParams.get('target') || '_top';

    useEffect(() => {
        history.push(historyUrl('widgets/workflow-status-badges/{namespace}', {namespace, name, label, target}));
    }, [namespace, name, label]);

    const [displayName, setDisplayName] = useState<string>();
    const [creationTimestamp, setCreationTimestamp] = useState<Date>(); // used to make sure we only display the most recent one
    const [phase, setPhase] = useState<NodePhase>('');

    useEffect(() => {
        const w = new RetryWatch(
            () => services.workflows.watch({namespace, name, labels: [label]}),
            () => setDisplayName(null),
            e => {
                const wf = e.object;
                const t = new Date(wf.metadata.creationTimestamp);
                if (t < creationTimestamp) {
                    return;
                }
                setDisplayName(wf.metadata.name);
                setPhase(wf.status.phase);
                setCreationTimestamp(t);
            },
            e => setDisplayName(e.message || 'error')
        );
        w.start();
        return () => w.stop();
    }, [namespace, name, label]);

    return (
        <a className='status-badge' href={uiUrl(`workflows/${namespace}/${displayName}`)} target={target}>
            <span className='label'>{displayName || 'not found'}</span>
            <span className={'status ' + phase}>{(phase || 'unknown').toLowerCase()} </span>
        </a>
    );
}
