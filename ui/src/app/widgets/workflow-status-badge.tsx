import * as React from 'react';
import {useEffect, useState} from 'react';
import {RouteComponentProps} from 'react-router';
import {NodePhase} from '../../models';
import {uiUrl} from '../shared/base';
import {historyUrl} from '../shared/history';
import {RetryWatch} from '../shared/retry-watch';
import {services} from '../shared/services';

require('./workflow-status-badge.scss');

export const WorkflowStatusBadge = ({history, match}: RouteComponentProps<any>) => {
    const [namespace] = useState(match.params.namespace);
    const [name] = useState(match.params.name);

    const queryParams = new URLSearchParams(location.search);

    const [target] = useState(queryParams.get('target') || '_top');

    useEffect(() => {
        history.push(historyUrl('widgets/workflow-status-badges/{namespace}/{name}', {namespace, name, target}));
    }, [namespace, name]);

    const [phase, setPhase] = useState<NodePhase>('');

    useEffect(() => {
        const w = new RetryWatch(
            () => services.workflows.watch({namespace, name}),
            () => {
                // noop
            },
            e => setPhase(e.object.status.phase),
            () => {
                // noop
            }
        );
        w.start();
        return () => w.stop();
    }, [namespace, name]);

    return (
        <a className='status-badge' href={uiUrl(`workflows/${namespace}/${name}`)} target={target}>
            <span className='label'>{name}</span>
            <span className={'status ' + phase}>{(phase || 'unknown').toLowerCase()} </span>
        </a>
    );
};
