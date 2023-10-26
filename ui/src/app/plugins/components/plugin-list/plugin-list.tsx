import {Page} from 'argo-ui';
import * as React from 'react';
import {useEffect, useState} from 'react';
import {RouteComponentProps} from 'react-router-dom';
import {uiUrl} from '../../../shared/base';
import {useCollectEvent} from '../../../shared/components/use-collect-event';
import {ZeroState} from '../../../shared/components/zero-state';
import {historyUrl} from '../../../shared/history';
import {Utils} from '../../../shared/utils';

export function PluginList({match, history}: RouteComponentProps<any>) {
    // state for URL and query parameters
    const [namespace] = useState(Utils.getNamespace(match.params.namespace) || '');
    useEffect(
        () =>
            history.push(
                historyUrl('plugins' + (Utils.managedNamespace ? '' : '/{namespace}'), {
                    namespace
                })
            ),
        [namespace]
    );
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
                    <a href='https://argoproj.github.io/argo-workflows/plugins/'>Learn more</a>.
                </p>
            </ZeroState>
        </Page>
    );
}
