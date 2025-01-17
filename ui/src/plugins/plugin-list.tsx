import {Page} from 'argo-ui/src/components/page/page';
import * as React from 'react';
import {useEffect, useRef, useState} from 'react';
import {RouteComponentProps} from 'react-router-dom';

import {uiUrl} from '../shared/base';
import {ZeroState} from '../shared/components/zero-state';
import {historyUrl} from '../shared/history';
import * as nsUtils from '../shared/namespaces';
import {useCollectEvent} from '../shared/use-collect-event';

export function PluginList({match, history}: RouteComponentProps<any>) {
    // state for URL and query parameters
    const isFirstRender = useRef(true);
    const [namespace] = useState(nsUtils.getNamespace(match.params.namespace) || '');
    useEffect(() => {
        if (isFirstRender.current) {
            isFirstRender.current = false;
            return;
        }
        history.push(
            historyUrl('plugins' + (nsUtils.getManagedNamespace() ? '' : '/{namespace}'), {
                namespace
            })
        );
    }, [namespace]);
    useCollectEvent('openedPlugins');

    return (
        <Page
            title='Plugins'
            toolbar={{
                breadcrumbs: [{title: 'Plugins', path: uiUrl('plugins')}]
            }}>
            <ZeroState title='Plugins'>
                <p>Plugins allow you to extend Argo Workflows with custom code.</p>
                <p>To list plugins:</p>
                <pre>kubectl get cm -l workflows.argoproj.io/configmap-type=ExecutorPlugin</pre>
                <br />
                <p>
                    <a href='https://argo-workflows.readthedocs.io/en/latest/plugins/'>Learn more</a>.
                </p>
            </ZeroState>
        </Page>
    );
}
