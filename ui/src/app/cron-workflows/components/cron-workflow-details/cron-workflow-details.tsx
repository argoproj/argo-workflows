import {NotificationType, Page, SlidingPanel} from 'argo-ui';
import * as React from 'react';
import {useContext, useEffect, useState} from 'react';
import {RouteComponentProps} from 'react-router';
import * as models from '../../../../models';
import {CronWorkflow, Link, Workflow} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {Loading} from '../../../shared/components/loading';
import {useCollectEvent} from '../../../shared/components/use-collect-event';
import {Context} from '../../../shared/context';
import {historyUrl} from '../../../shared/history';
import {Pagination, parseLimit} from '../../../shared/pagination';
import {services} from '../../../shared/services';
import {useQueryParams} from '../../../shared/use-query-params';
import * as Actions from '../../../shared/workflow-operations-map';
import {WidgetGallery} from '../../../widgets/widget-gallery';
import {allBatchActionsEnabled, WorkflowListRenderOptions} from '../../../workflows/components/workflows-list/workflows-list';
import {WorkflowsRow} from '../../../workflows/components/workflows-row/workflows-row';
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
    const [workflows, setWorkflows] = useState<Workflow[]>([]);
    const [{selectedPhases, selectedLabels, paginationLimit}, setSelectedData] = useState<WorkflowListRenderOptions>(JSON.parse(localStorage.getItem('ListOptions/options')));
    const [, setBatchActionDisabled] = useState<Actions.OperationDisabled>();
    const [selectedWorkflows, setSelectedWorkflows] = useState(new Map<string, models.Workflow>());
    const [columns, setColumns] = useState<models.Column[]>([]);
    const [pagination] = useState<Pagination>({offset: queryParams.get('offset'), limit: parseLimit(queryParams.get('limit') || `${paginationLimit}` || '50')});

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

    useEffect(() => {
        services.workflows.list(namespace, selectedPhases, selectedLabels, pagination).then(res => {
            setWorkflows(res.items.filter(el => el.metadata.name.substring(0, el.metadata.name.lastIndexOf('-')) === name));
        });

        services.info.getInfo().then(info => {
            setColumns(info.columns);
        });
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
    const updateCurrentlySelectedAndBatchActions = (newSelectedWorkflows: Map<string, Workflow>): void => {
        const actions: any = Actions.WorkflowOperationsMap;
        const nowDisabled: any = {...allBatchActionsEnabled};
        for (const action of Object.keys(nowDisabled)) {
            for (const wf of Array.from(newSelectedWorkflows.values())) {
                nowDisabled[action] = nowDisabled[action] || actions[action].disabled(wf);
            }
        }
        setBatchActionDisabled(nowDisabled);
        setSelectedWorkflows(new Map<string, models.Workflow>(newSelectedWorkflows));
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
                <div className='argo-table-list'>
                    <div className='row argo-table-list__head'>
                        <div className='columns small-1 workflows-list__status'>
                            <input
                                type='checkbox'
                                className='workflows-list__status--checkbox'
                                checked={workflows.length === selectedWorkflows.size}
                                onClick={e => {
                                    e.stopPropagation();
                                }}
                                onChange={e => {
                                    if (workflows.length === selectedWorkflows.size) {
                                        // All workflows are selected, deselect them all
                                        updateCurrentlySelectedAndBatchActions(new Map<string, models.Workflow>());
                                    } else {
                                        // Not all workflows are selected, select them all
                                        const currentlySelected: Map<string, Workflow> = selectedWorkflows;
                                        workflows.forEach(wf => {
                                            if (!currentlySelected.has(wf.metadata.uid)) {
                                                currentlySelected.set(wf.metadata.uid, wf);
                                            }
                                        });
                                        updateCurrentlySelectedAndBatchActions(currentlySelected);
                                    }
                                }}
                            />
                        </div>
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
                    {workflows.map(wf => {
                        return (
                            <WorkflowsRow
                                workflow={wf}
                                key={wf.metadata.uid}
                                checked={selectedWorkflows.has(wf.metadata.uid)}
                                columns={columns}
                                onChange={key => {
                                    const value = `${key}=${wf.metadata?.labels[key]}`;
                                    let newTags: string[] = [];
                                    if (selectedLabels.indexOf(value) === -1) {
                                        newTags = selectedLabels.concat(value);
                                        setSelectedData(prev => ({...prev, selectedLabels: newTags}));
                                    }
                                }}
                                select={subWf => {
                                    const wfUID = subWf.metadata.uid;
                                    if (!wfUID) {
                                        return;
                                    }
                                    const currentlySelected: Map<string, Workflow> = selectedWorkflows;
                                    if (!currentlySelected.has(wfUID)) {
                                        currentlySelected.set(wfUID, subWf);
                                    } else {
                                        currentlySelected.delete(wfUID);
                                    }
                                    updateCurrentlySelectedAndBatchActions(currentlySelected);
                                }}
                            />
                        );
                    })}
                </div>
            </>
        </Page>
    );
};
