import {NotificationType} from 'argo-ui/src/components/notifications/notifications';
import {Page} from 'argo-ui/src/components/page/page';
import {SlidingPanel} from 'argo-ui/src/components/sliding-panel/sliding-panel';
import * as React from 'react';
import {useContext, useEffect, useState} from 'react';
import {RouteComponentProps} from 'react-router';

import * as models from '../../models';
import {CronWorkflow, Link, Workflow} from '../../models';
import {uiUrl} from '../shared/base';
import {ErrorNotice} from '../shared/components/error-notice';
import {openLinkWithKey} from '../shared/components/links';
import {Loading} from '../shared/components/loading';
import {useCollectEvent} from '../shared/use-collect-event';
import {ZeroState} from '../shared/components/zero-state';
import {Context} from '../shared/context';
import {historyUrl} from '../shared/history';
import {services} from '../shared/services';
import {useQueryParams} from '../shared/use-query-params';
import {useEditableResource} from '../shared/use-editable-resource';
import {WidgetGallery} from '../widgets/widget-gallery';
import {WorkflowDetailsList} from '../workflows/components/workflow-details-list/workflow-details-list';
import {CronWorkflowEditor} from './cron-workflow-editor';

import '../workflows/components/workflow-details/workflow-details.scss';

export function CronWorkflowDetails({match, location, history}: RouteComponentProps<any>) {
    // boiler-plate
    const {navigation, notifications, popup} = useContext(Context);
    const queryParams = new URLSearchParams(location.search);

    const [namespace] = useState(match.params.namespace);
    const [name] = useState(match.params.name);
    const [sidePanel, setSidePanel] = useState(queryParams.get('sidePanel'));
    const [tab, setTab] = useState(queryParams.get('tab'));
    const [workflows, setWorkflows] = useState<Workflow[]>([]);
    const [columns, setColumns] = useState<models.Column[]>([]);

    const [cronWorkflow, edited, setCronWorkflow, resetCronWorkflow] = useEditableResource<CronWorkflow>();
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
                historyUrl('cron-workflows/{namespace}/{name}', {
                    namespace,
                    name,
                    sidePanel,
                    tab
                })
            ),
        [namespace, name, sidePanel, tab]
    );

    useEffect(() => {
        services.cronWorkflows
            .get(name, namespace)
            .then(resetCronWorkflow)
            .then(() => setError(null))
            .catch(setError);
    }, [namespace, name]);

    useEffect(() => {
        (async () => {
            const workflowList = await services.workflows.list(namespace, null, [`${models.labels.cronWorkflow}=${name}`], {limit: 50});
            const workflowsInfo = await services.info.getInfo();

            setWorkflows(workflowList.items);
            setColumns(workflowsInfo.columns);
        })();
    }, []);

    useCollectEvent('openedCronWorkflowDetails');

    const suspendButton =
        cronWorkflow && !cronWorkflow.spec.suspend
            ? {
                  title: 'Suspend',
                  iconClassName: 'fa fa-pause',
                  action: () =>
                      services.cronWorkflows
                          .suspend(name, namespace)
                          .then(resetCronWorkflow)
                          .then(() => setError(null))
                          .catch(setError),
                  disabled: !cronWorkflow || edited
              }
            : {
                  title: 'Resume',
                  iconClassName: 'fa fa-play',
                  action: () =>
                      services.cronWorkflows
                          .resume(name, namespace)
                          .then(resetCronWorkflow)
                          .then(() => setError(null))
                          .catch(setError),
                  disabled: !cronWorkflow || !cronWorkflow.spec.suspend || edited
              };

    const getItems = () => {
        const items = [
            {
                title: 'Submit',
                iconClassName: 'fa fa-plus',
                disabled: edited,
                action: () =>
                    services.workflows
                        .submit('cronwf', name, namespace)
                        .then(wf => navigation.goto(uiUrl(`workflows/${wf.metadata.namespace}/${wf.metadata.name}`)))
                        .then(() => setError(null))
                        .catch(setError)
            },
            {
                title: 'Update',
                iconClassName: 'fa fa-save',
                disabled: !edited,
                action: () => {
                    // magic - we get the latest from the server and then apply the changes from the rendered version to this
                    return services.cronWorkflows
                        .get(name, namespace)
                        .then(latest =>
                            services.cronWorkflows.update(
                                {
                                    ...latest,
                                    spec: cronWorkflow.spec,
                                    metadata: {...cronWorkflow.metadata, resourceVersion: latest.metadata.resourceVersion}
                                },
                                cronWorkflow.metadata.name,
                                cronWorkflow.metadata.namespace
                            )
                        )
                        .then(resetCronWorkflow)
                        .then(() => notifications.show({content: 'Updated', type: NotificationType.Success}))
                        .then(() => setError(null))
                        .catch(setError);
                }
            },
            suspendButton,
            {
                title: 'Delete',
                iconClassName: 'fa fa-trash',
                disabled: edited,
                action: () => {
                    popup.confirm('confirm', 'Are you sure you want to delete this cron workflow?').then(yes => {
                        if (yes) {
                            services.cronWorkflows
                                .delete(name, namespace)
                                .then(() => navigation.goto(uiUrl('cron-workflows/' + namespace)))
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
        ];

        if (cronWorkflow?.spec?.workflowSpec?.workflowTemplateRef) {
            const templateName = cronWorkflow.spec.workflowSpec.workflowTemplateRef.name;
            const clusterScope = cronWorkflow.spec.workflowSpec.workflowTemplateRef.clusterScope;
            const url: string = clusterScope ? uiUrl(`cluster-workflow-templates/${templateName}`) : uiUrl(`workflow-templates/${cronWorkflow.metadata.namespace}/${templateName}`);
            const icon: string = clusterScope ? 'fa fa-window-restore' : 'fa fa-window-maximize';

            const templateLink: Link = {
                name: 'Open Workflow Template',
                scope: 'workflow',
                url
            };

            items.push({
                title: templateLink.name,
                iconClassName: icon,
                action: () => openLinkWithKey(templateLink.url)
            });
        }

        return items;
    };

    return (
        <Page
            title='Cron Workflow Details'
            toolbar={{
                breadcrumbs: [
                    {title: 'Cron Workflows', path: uiUrl('cron-workflows')},
                    {title: namespace, path: uiUrl('cron-workflows/' + namespace)},
                    {title: name, path: uiUrl('cron-workflows/' + namespace + '/' + name)}
                ],
                actionMenu: {
                    items: getItems()
                }
            }}>
            <>
                <ErrorNotice error={error} />
                {!cronWorkflow ? (
                    <Loading />
                ) : (
                    <CronWorkflowEditor cronWorkflow={cronWorkflow} onChange={setCronWorkflow} onError={setError} selectedTabKey={tab} onTabSelected={setTab} />
                )}
                <SlidingPanel isShown={!!sidePanel} onClose={() => setSidePanel(null)}>
                    {sidePanel === 'share' && <WidgetGallery namespace={namespace} label={'workflows.argoproj.io/cron-workflow=' + name} />}
                </SlidingPanel>
                <>
                    <ErrorNotice error={error} />
                    {!workflows ? (
                        <ZeroState title='No completed cron workflows'>
                            <p> You can create new cron workflows here or using the CLI. </p>
                        </ZeroState>
                    ) : (
                        <WorkflowDetailsList workflows={workflows} columns={columns} />
                    )}
                </>
            </>
        </Page>
    );
}
