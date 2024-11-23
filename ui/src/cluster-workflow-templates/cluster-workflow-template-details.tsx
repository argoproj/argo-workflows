import {NotificationType} from 'argo-ui/src/components/notifications/notifications';
import {Page} from 'argo-ui/src/components/page/page';
import {SlidingPanel} from 'argo-ui/src/components/sliding-panel/sliding-panel';
import * as React from 'react';
import {useContext, useEffect, useState} from 'react';
import {RouteComponentProps} from 'react-router';

import {uiUrl} from '../shared/base';
import {ErrorNotice} from '../shared/components/error-notice';
import {Loading} from '../shared/components/loading';
import {ZeroState} from '../shared/components/zero-state';
import {Context} from '../shared/context';
import {historyUrl} from '../shared/history';
import * as models from '../shared/models';
import {ClusterWorkflowTemplate, Workflow} from '../shared/models';
import * as nsUtils from '../shared/namespaces';
import {services} from '../shared/services';
import {useCollectEvent} from '../shared/use-collect-event';
import {useEditableObject} from '../shared/use-editable-object';
import {useQueryParams} from '../shared/use-query-params';
import {WorkflowTemplateEditor} from '../workflow-templates/workflow-template-editor';
import {SubmitWorkflowPanel} from '../workflows/components/submit-workflow-panel';
import {WorkflowDetailsList} from '../workflows/components/workflow-details-list/workflow-details-list';

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
    const {object: template, setObject: setTemplate, resetObject: resetTemplate, serialization, edited, lang, setLang} = useEditableObject<ClusterWorkflowTemplate>();

    useEffect(
        useQueryParams(history, p => {
            setSidePanel(p.get('sidePanel') === 'true');
            setTab(p.get('tab'));
        }),
        [history]
    );

    useEffect(() => {
        history.push(historyUrl('cluster-workflow-templates/{name}', {name, sidePanel, tab}));
    }, [name, sidePanel, tab]);

    useEffect(() => {
        (async () => {
            try {
                const newTemplate = await services.clusterWorkflowTemplate.get(name);
                resetTemplate(newTemplate);
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
                setNamespace(nsUtils.getNamespaceWithDefault(info.managedNamespace));
                setError(null);
            } catch (err) {
                setError(err);
            }
        })();
    }, []);

    useCollectEvent('openedClusterWorkflowTemplateDetails');

    return (
        <Page
            title='Cluster Workflow Template Details'
            toolbar={{
                breadcrumbs: [
                    {title: 'Cluster Workflow Templates', path: uiUrl('cluster-workflow-templates')},
                    {title: name, path: uiUrl('cluster-workflow-templates/' + name)}
                ],
                actionMenu: {
                    items: [
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
                                    .then(resetTemplate)
                                    .then(() =>
                                        notifications.show({
                                            content: 'Updated',
                                            type: NotificationType.Success
                                        })
                                    )
                                    .then(() => setError(null))
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
                    ]
                }
            }}>
            <>
                <ErrorNotice error={error} />
                {!template ? (
                    <Loading />
                ) : (
                    <WorkflowTemplateEditor
                        template={template}
                        serialization={serialization}
                        lang={lang}
                        onLangChange={setLang}
                        onChange={setTemplate}
                        onError={setError}
                        onTabSelected={setTab}
                        selectedTabKey={tab}
                    />
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
                    <WorkflowDetailsList workflows={workflows} columns={columns} />
                )}
            </>
        </Page>
    );
}
