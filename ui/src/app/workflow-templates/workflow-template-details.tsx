import {NotificationType} from 'argo-ui/src/components/notifications/notifications';
import {Page} from 'argo-ui/src/components/page/page';
import {SlidingPanel} from 'argo-ui/src/components/sliding-panel/sliding-panel';
import * as React from 'react';
import {useContext, useEffect, useState} from 'react';
import {RouteComponentProps} from 'react-router';

import * as models from '../../models';
import {WorkflowTemplate, Workflow} from '../../models';
import {uiUrl} from '../shared/base';
import {ErrorNotice} from '../shared/components/error-notice';
import {Loading} from '../shared/components/loading';
import {useEditableObject} from '../shared/use-editable-object';
import {useCollectEvent} from '../shared/use-collect-event';
import {ZeroState} from '../shared/components/zero-state';
import {Context} from '../shared/context';
import {historyUrl} from '../shared/history';
import {services} from '../shared/services';
import {useQueryParams} from '../shared/use-query-params';
import {WidgetGallery} from '../widgets/widget-gallery';
import {WorkflowDetailsList} from '../workflows/components/workflow-details-list/workflow-details-list';
import {SubmitWorkflowPanel} from '../workflows/components/submit-workflow-panel';
import {WorkflowTemplateEditor} from './workflow-template-editor';

export function WorkflowTemplateDetails({history, location, match}: RouteComponentProps<any>) {
    // boiler-plate
    const {notifications, navigation, popup} = useContext(Context);
    const queryParams = new URLSearchParams(location.search);

    // state for URL and query parameters
    const namespace = match.params.namespace;
    const name = match.params.name;
    const [sidePanel, setSidePanel] = useState(queryParams.get('sidePanel'));
    const [tab, setTab] = useState<string>(queryParams.get('tab'));
    const [workflows, setWorkflows] = useState<Workflow[]>([]);
    const [columns, setColumns] = useState<models.Column[]>([]);

    const [template, edited, setTemplate, resetTemplate] = useEditableObject<WorkflowTemplate>();
    const [error, setError] = useState<Error>();

    useEffect(
        useQueryParams(history, p => {
            setSidePanel(p.get('sidePanel'));
            setTab(p.get('tab'));
        }),
        [history]
    );

    useEffect(
        () =>
            history.push(
                historyUrl('workflow-templates/{namespace}/{name}', {
                    namespace,
                    name,
                    sidePanel,
                    tab
                })
            ),
        [namespace, name, sidePanel, tab]
    );

    useEffect(() => {
        services.workflowTemplate
            .get(name, namespace)
            .then(resetTemplate)
            .then(() => setError(null))
            .catch(setError);
    }, [name, namespace]);

    useEffect(() => {
        (async () => {
            const workflowList = await services.workflows.list(namespace, null, [`${models.labels.workflowTemplate}=${name}`], {limit: 50});
            const workflowsInfo = await services.info.getInfo();

            setWorkflows(workflowList.items);
            setColumns(workflowsInfo.columns);
        })();
    }, []);

    useCollectEvent('openedWorkflowTemplateDetails');

    return (
        <Page
            title='Workflow Template Details'
            toolbar={{
                breadcrumbs: [
                    {title: 'Workflow Templates', path: uiUrl('workflow-templates')},
                    {title: namespace, path: uiUrl('workflow-templates/' + namespace)},
                    {title: name, path: uiUrl('workflow-templates/' + namespace + '/' + name)}
                ],
                actionMenu: {
                    items: [
                        {
                            title: 'Submit',
                            iconClassName: 'fa fa-plus',
                            disabled: edited,
                            action: () => setSidePanel('submit')
                        },
                        {
                            title: 'Update',
                            iconClassName: 'fa fa-save',
                            disabled: !edited,
                            action: () =>
                                services.workflowTemplate
                                    .update(template, name, namespace)
                                    .then(resetTemplate)
                                    .then(() => notifications.show({content: 'Updated', type: NotificationType.Success}))
                                    .then(() => setError(null))
                                    .catch(setError)
                        },
                        {
                            title: 'Delete',
                            iconClassName: 'fa fa-trash',
                            action: () => {
                                popup.confirm('confirm', 'Are you sure you want to delete this workflow template?').then(yes => {
                                    if (yes) {
                                        services.workflowTemplate
                                            .delete(name, namespace)
                                            .then(() => navigation.goto(uiUrl('workflow-templates/' + namespace)))
                                            .then(() => setError(null))
                                            .catch(setError);
                                    }
                                });
                            }
                        },
                        {
                            title: 'Share',
                            iconClassName: 'fa fa-share-alt',
                            action: () => setSidePanel('share')
                        }
                    ]
                }
            }}>
            <>
                <ErrorNotice error={error} />
                {!template ? <Loading /> : <WorkflowTemplateEditor template={template} onChange={setTemplate} onError={setError} onTabSelected={setTab} selectedTabKey={tab} />}
            </>
            {template && (
                <SlidingPanel isShown={!!sidePanel} onClose={() => setSidePanel(null)} isMiddle={sidePanel === 'submit'}>
                    {sidePanel === 'submit' && (
                        <SubmitWorkflowPanel
                            kind='WorkflowTemplate'
                            namespace={namespace}
                            name={name}
                            entrypoint={template.spec.entrypoint}
                            templates={template.spec.templates || []}
                            workflowParameters={template.spec.arguments.parameters || []}
                        />
                    )}
                    {sidePanel === 'share' && <WidgetGallery namespace={namespace} label={'workflows.argoproj.io/workflow-template=' + name} />}
                </SlidingPanel>
            )}
            <>
                <ErrorNotice error={error} />
                {!workflows ? (
                    <ZeroState title='No completed workflow templates'>
                        <p> You can create new workflow templates here or using the CLI. </p>
                    </ZeroState>
                ) : (
                    <WorkflowDetailsList workflows={workflows} columns={columns} />
                )}
            </>
        </Page>
    );
}
