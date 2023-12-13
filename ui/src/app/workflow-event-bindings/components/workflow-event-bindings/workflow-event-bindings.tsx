import {Page, SlidingPanel} from 'argo-ui';
import * as React from 'react';
import {useContext, useEffect, useState} from 'react';
import {RouteComponentProps} from 'react-router-dom';
import {WorkflowEventBinding} from '../../../../models';
import {absoluteUrl, uiUrl} from '../../../shared/base';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {InfoIcon} from '../../../shared/components/fa-icons';
import {GraphPanel} from '../../../shared/components/graph/graph-panel';
import {Graph} from '../../../shared/components/graph/types';
import {Loading} from '../../../shared/components/loading';
import {NamespaceFilter} from '../../../shared/components/namespace-filter';
import {ResourceEditor} from '../../../shared/components/resource-editor/resource-editor';
import {useCollectEvent} from '../../../shared/components/use-collect-event';
import {ZeroState} from '../../../shared/components/zero-state';
import {Context} from '../../../shared/context';
import {Footnote} from '../../../shared/footnote';
import {historyUrl} from '../../../shared/history';
import {services} from '../../../shared/services';
import {useQueryParams} from '../../../shared/use-query-params';
import {Utils} from '../../../shared/utils';
import {ID} from './id';

const introductionText = (
    <>
        Workflow event bindings allow you to trigger workflows when a webhook event is received. For example, start a build on a Git commit, or start a machine learning pipeline
        from a remote system.
    </>
);
const learnMore = <a href={'https://argoproj.github.io/argo-workflows/events/'}>Learn more</a>;

export function WorkflowEventBindings({match, location, history}: RouteComponentProps<any>) {
    // boiler-plate
    const ctx = useContext(Context);
    const queryParams = new URLSearchParams(location.search);

    // state for URL and query parameters
    const [namespace, setNamespace] = useState(Utils.getNamespace(match.params.namespace) || '');
    const [selectedWorkflowEventBinding, setSelectedWorkflowEventBinding] = useState(queryParams.get('selectedWorkflowEventBinding'));

    useEffect(
        useQueryParams(history, p => {
            setSelectedWorkflowEventBinding(p.get('selectedWorkflowEventBinding'));
        }),
        [history]
    );

    useEffect(
        () =>
            history.push(
                historyUrl('workflow-event-bindings' + (Utils.managedNamespace ? '' : '/{namespace}'), {
                    namespace,
                    selectedWorkflowEventBinding
                })
            ),
        [namespace, selectedWorkflowEventBinding]
    );

    // internal state
    const [error, setError] = useState<Error>();
    const [workflowEventBindings, setWorkflowEventBindings] = useState<WorkflowEventBinding[]>();

    const selected = (workflowEventBindings || []).find(x => x.metadata.namespace + '/' + x.metadata.name === selectedWorkflowEventBinding);

    const g = new Graph();
    (workflowEventBindings || []).forEach(web => {
        const bindingId = ID.join('WorkflowEventBinding', web.metadata.namespace, web.metadata.name);
        g.nodes.set(bindingId, {label: web.spec.event.selector, genre: 'event', icon: 'cloud'});
        if (web.spec.submit) {
            const x = web.spec.submit.workflowTemplateRef;
            const templateId = ID.join(x.clusterScope ? 'ClusterWorkflowTemplate' : 'WorkflowTemplate', web.metadata.namespace, x.name);
            g.nodes.set(templateId, {
                label: x.name,
                genre: x.clusterScope ? 'cluster-template' : 'template',
                icon: x.clusterScope ? 'window-restore' : 'window-maximize'
            });
            g.edges.set({v: bindingId, w: templateId}, {});
        }
    });

    useEffect(() => {
        services.event
            .listWorkflowEventBindings(namespace)
            .then(list => setWorkflowEventBindings(list.items || []))
            .then(() => setError(null))
            .catch(setError);
    }, [namespace]);

    useCollectEvent('openedWorkflowEventBindings');

    return (
        <Page
            title='Workflow Event Bindings'
            toolbar={{
                breadcrumbs: [
                    {title: 'Workflow Event Bindings', path: uiUrl('workflow-event-bindings')},
                    {title: namespace, path: uiUrl('workflow-event-bindings/' + namespace)}
                ],
                tools: [<NamespaceFilter key='namespace-filter' value={namespace} onChange={setNamespace} />]
            }}>
            <ErrorNotice error={error} />
            {!workflowEventBindings ? (
                <Loading />
            ) : workflowEventBindings.length === 0 ? (
                <ZeroState>
                    <p>{introductionText}</p>
                    <p>
                        Once you&apos;ve created a workflow event binding, you can test it from the CLI using <code>curl</code>, for example:
                    </p>
                    <p>
                        <code>
                            curl &apos;{absoluteUrl('api/v1/events/{namespace}/-')}&apos; -H &apos;Content-Type: application/json&apos; -H &apos;Authorization: $ARGO_TOKEN&apos; -d
                            &apos;&#123;&#125;&apos;
                        </code>
                    </p>
                    <p>
                        You&apos;ll probably find it easiest to experiment and test using the <a href={uiUrl('apidocs')}>graphical interface to the API </a> - look for
                        &quot;EventService&quot;.
                    </p>
                    <p>{learnMore}</p>
                </ZeroState>
            ) : (
                <>
                    <GraphPanel
                        storageScope='workflow-event-bindings'
                        graph={g}
                        nodeGenresTitle={'Type'}
                        nodeGenres={{'event': true, 'template': true, 'cluster-template': true}}
                        horizontal={true}
                        onNodeSelect={id => {
                            const x = ID.split(id);
                            if (x.type === 'ClusterWorkflowTemplate') {
                                ctx.navigation.goto(uiUrl('cluster-workflow-templates/' + x.name));
                            } else if (x.type === 'WorkflowTemplate') {
                                ctx.navigation.goto(uiUrl('workflow-templates/' + x.namespace + '/' + x.name));
                            } else {
                                setSelectedWorkflowEventBinding(x.namespace + '/' + x.name);
                            }
                        }}
                    />
                    <Footnote>
                        <InfoIcon /> {introductionText} {learnMore}.
                    </Footnote>
                    <SlidingPanel isShown={!!selectedWorkflowEventBinding} onClose={() => setSelectedWorkflowEventBinding(null)}>
                        {selected && <ResourceEditor value={selected} />}
                    </SlidingPanel>
                </>
            )}
        </Page>
    );
}
