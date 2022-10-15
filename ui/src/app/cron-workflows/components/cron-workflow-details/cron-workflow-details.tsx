import {NotificationType, Page, SlidingPanel} from 'argo-ui';
import * as React from 'react';
import {useContext, useEffect, useState} from 'react';
import {RouteComponentProps} from 'react-router';
import {CronWorkflow, Link} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {Loading} from '../../../shared/components/loading';
import {useCollectEvent} from '../../../shared/components/use-collect-event';
import {Context} from '../../../shared/context';
import {historyUrl} from '../../../shared/history';
import {services} from '../../../shared/services';
import {useQueryParams} from '../../../shared/use-query-params';
import {WidgetGallery} from '../../../widgets/widget-gallery';
import {CronWorkflowEditor} from '../cron-workflow-editor';

require('../../../workflows/components/workflow-details/workflow-details.scss');

export const CronWorkflowDetails = ({match, location, history}: RouteComponentProps<any>) => {
    // boiler-plate
    const {navigation, notifications, popup} = useContext(Context);
    const queryParams = new URLSearchParams(location.search);

    const [namespace] = useState(match.params.namespace);
    const [name] = useState(match.params.name);
    const [sidePanel, setSidePanel] = useState(queryParams.get('sidePanel'));
    const [tab, setTab] = useState(queryParams.get('tab'));

    const [cronWorkflow, setCronWorkflow] = useState<CronWorkflow>();
    const [edited, setEdited] = useState(false);
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
            .then(setCronWorkflow)
            .then(() => setEdited(false))
            .then(() => setError(null))
            .catch(setError);
    }, [namespace, name]);

    useEffect(() => setEdited(true), [cronWorkflow]);

    useCollectEvent('openedCronWorkflowDetails');

    const suspendButton =
        cronWorkflow && !cronWorkflow.spec.suspend
            ? {
                  title: 'Suspend',
                  iconClassName: 'fa fa-pause',
                  action: () =>
                      services.cronWorkflows
                          .suspend(name, namespace)
                          .then(setCronWorkflow)
                          .then(() => setEdited(false))
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
                          .then(setCronWorkflow)
                          .then(() => setEdited(false))
                          .then(() => setError(null))
                          .catch(setError),
                  disabled: !cronWorkflow || !cronWorkflow.spec.suspend || edited
              };

    const openLink = (link: Link) => {
        if ((window.event as MouseEvent).ctrlKey || (window.event as MouseEvent).metaKey) {
            window.open(link.url, '_blank');
        } else {
            document.location.href = link.url;
        }
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
                        .then(setCronWorkflow)
                        .then(() => notifications.show({content: 'Updated', type: NotificationType.Success}))
                        .then(() => setError(null))
                        .then(() => setEdited(false))
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
                action: () => openLink(templateLink)
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
            </>
        </Page>
    );
};
