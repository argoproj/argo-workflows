import {NotificationType} from 'argo-ui/src/components/notifications/notifications';
import {Page} from 'argo-ui/src/components/page/page';
import {SlidingPanel} from 'argo-ui/src/components/sliding-panel/sliding-panel';
import * as React from 'react';
import {useContext, useEffect, useState} from 'react';
import {RouteComponentProps} from 'react-router';

import * as models from '../../models';
import {ClusterWorkflowTemplate, Workflow} from '../../models';
import {uiUrl} from '../shared/base';
import {ErrorNotice} from '../shared/components/error-notice';
import {Loading} from '../shared/components/loading';
import {useCollectEvent} from '../shared/use-collect-event';
import {ZeroState} from '../shared/components/zero-state';
import {Context} from '../shared/context';
import {historyUrl} from '../shared/history';
import {services} from '../shared/services';
import {useQueryParams} from '../shared/use-query-params';
import {Utils} from '../shared/utils';
import {WorkflowsRow} from '../workflows/components/workflows-row/workflows-row';
import {SubmitWorkflowPanel} from '../workflows/components/submit-workflow-panel';
import {ClusterWorkflowTemplateEditor} from './cluster-workflow-template-editor';

import '../workflows/components/workflow-details/workflow-details.scss';

export function ClusterWorkflowTemplateDetails({history, location, match}: RouteComponentProps<any>) {
    // boiler-plate
    const {navigation, notifications, popup} = useContext(Context);
    const queryParams = new URLSearchParams(location.search);

    const name = match.params.name;
    const [namespace, setNamespace] = useState<string>();
    const [sidePanel, setSidePanel] = useState(queryParams.get('sidePanel') === 'true');
    const [tab, setTab] = useState<string>(queryParams.get('tab'));
    const [workflows, setWorkflows] = useState<Workflow[]>([]);
    const [columns, setColumns] = useState<models.Column[]>([]);

    const [error, setError] = useState<Error>();
    const [template, setTemplate] = useState<ClusterWorkflowTemplate>();
    const [edited, setEdited] = useState(false);

    useEffect(
        useQueryParams(history, p => {
            setSidePanel(p.get('sidePanel') === 'true');
            setTab(p.get('tab'));
        }),
        [history]
    );

    useEffect(() => setEdited(true), [template]);
    useEffect(() => {
        history.push(historyUrl('cluster-workflow-templates/{name}', {name, sidePanel, tab}));
    }, [name, sidePanel, tab]);

    useEffect(() => {
        (async () => {
            try {
                const newTemplate = await services.clusterWorkflowTemplate.get(name);
                setTemplate(newTemplate);
                setEdited(false); // set back to false
                setError(null);
            } catch (err) {
                setError(err);
            }
        })();
    }, [name]);

    useEffect(() => {
        (async () => {
            try {
                const workflowList = await services.workflows.list('', null, [`${models.labels.clusterWorkflowTemplate}=${name}`], {limit: 50});
                const info = await services.info.getInfo();

                setWorkflows(workflowList.items);
                setColumns(info.columns);
                setNamespace(Utils.getNamespaceWithDefault(info.managedNamespace));
                setError(null);
            } catch (err) {
                setError(err);
            }
        })();
    }, []);

    useCollectEvent('openedClusterWorkflowTemplateDetails');

    const getItems = () => {
        const items = [
            {
                title: 'Submit',
                iconClassName: 'fa fa-plus',
                disabled: edited,
                action: () => setSidePanel(true)
            },
            {
                title: 'Update',
                iconClassName: 'fa fa-save',
                disabled: !edited,
                action: () => {
                    services.clusterWorkflowTemplate
                        .update(template, name)
                        .then(setTemplate)
                        .then(() =>
                            notifications.show({
                                content: 'Updated',
                                type: NotificationType.Success
                            })
                        )
                        .then(() => setError(null))
                        .then(() => setEdited(false))
                        .catch(setError);
                }
            },
            {
                title: 'Delete',
                iconClassName: 'fa fa-trash',
                action: () => {
                    popup.confirm('confirm', 'Are you sure you want to delete this cluster workflow template?').then(yes => {
                        if (yes) {
                            services.clusterWorkflowTemplate
                                .delete(name)
                                .then(() => navigation.goto(uiUrl('cluster-workflow-templates')))
                                .then(() => setError(null))
                                .catch(setError);
                        }
                    });
                }
            }
        ];

        return items;
    };

    return (
        <Page
            title='Cluster Workflow Template Details'
            toolbar={{
                breadcrumbs: [
                    {title: 'Cluster Workflow Templates', path: uiUrl('cluster-workflow-templates')},
                    {title: name, path: uiUrl('cluster-workflow-templates/' + name)}
                ],
                actionMenu: {
                    items: getItems()
                }
            }}>
            <>
                <>
                    <ErrorNotice error={error} />
                    {!template ? (
                        <Loading />
                    ) : (
                        <ClusterWorkflowTemplateEditor template={template} onChange={setTemplate} onError={setError} onTabSelected={setTab} selectedTabKey={tab} />
                    )}
                </>
                {template && (
                    <SlidingPanel isShown={!!sidePanel} onClose={() => setSidePanel(null)} isMiddle={true}>
                        <SubmitWorkflowPanel
                            kind='ClusterWorkflowTemplate'
                            namespace={namespace}
                            name={template.metadata.name}
                            entrypoint={template.spec.entrypoint}
                            templates={template.spec.templates || []}
                            workflowParameters={template.spec.arguments.parameters || []}
                        />
                    </SlidingPanel>
                )}
                <>
                    <ErrorNotice error={error} />
                    {!workflows ? (
                        <ZeroState title='No completed cluster workflow templates'>
                            <p> You can create new cluster workflow templates here or using the CLI. </p>
                        </ZeroState>
                    ) : (
                        <div className='argo-table-list workflows-cluster-template-list'>
                            <div className='row argo-table-list__head'>
                                <div className='columns small-1 workflows-list__status' />
                                <div className='row small-11'>
                                    <div className='columns small-2'>NAME</div>
                                    <div className='columns small-1'>NAMESPACE</div>
                                    <div className='columns small-1'>STARTED</div>
                                    <div className='columns small-1'>FINISHED</div>
                                    <div className='columns small-1'>DURATION</div>
                                    <div className='columns small-1'>PROGRESS</div>
                                    <div className='columns small-2'>MESSAGE</div>
                                    <div className='columns small-1'>DETAILS</div>
                                    <div className='columns small-1'>ARCHIVED</div>
                                    {(columns || []).map(col => {
                                        return (
                                            <div className='columns small-1' key={col.key}>
                                                {col.name}
                                            </div>
                                        );
                                    })}
                                </div>
                            </div>
                            {/* checkboxes are not visible and are unused on this page */}
                            {workflows.map(wf => {
                                return <WorkflowsRow workflow={wf} key={wf.metadata.uid} checked={false} columns={columns} onChange={null} select={null} />;
                            })}
                        </div>
                    )}
                </>
            </>
        </Page>
    );
}
