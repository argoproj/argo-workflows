import {Page, SlidingPanel} from 'argo-ui';
import * as React from 'react';
import {useContext, useEffect, useRef, useState} from 'react';
import {RouteComponentProps} from 'react-router-dom';
import * as models from '../../../../models';
import {labels, Workflow, WorkflowPhase, WorkflowPhases} from '../../../../models';
import {uiUrl} from '../../../shared/base';

import {CostOptimisationNudge} from '../../../shared/components/cost-optimisation-nudge';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {ExampleManifests} from '../../../shared/components/example-manifests';
import {Loading} from '../../../shared/components/loading';
import {PaginationPanel} from '../../../shared/components/pagination-panel';
import {useCollectEvent} from '../../../shared/components/use-collect-event';
import {ZeroState} from '../../../shared/components/zero-state';
import {Context} from '../../../shared/context';
import {historyUrl} from '../../../shared/history';
import {ListWatch, sortByYouth} from '../../../shared/list-watch';
import {Pagination, parseLimit} from '../../../shared/pagination';
import {ScopedLocalStorage} from '../../../shared/scoped-local-storage';
import {services} from '../../../shared/services';
import {Utils} from '../../../shared/utils';
import * as Actions from '../../../shared/workflow-operations-map';
import {WorkflowCreator} from '../workflow-creator';
import {WorkflowFilters} from '../workflow-filters/workflow-filters';
import {WorkflowsRow} from '../workflows-row/workflows-row';
import {WorkflowsSummaryContainer} from '../workflows-summary-container/workflows-summary-container';
import {WorkflowsToolbar} from '../workflows-toolbar/workflows-toolbar';

require('./workflows-list.scss');

interface WorkflowListRenderOptions {
    paginationLimit: number;
    selectedPhases: WorkflowPhase[];
    selectedLabels: string[];
}

const allBatchActionsEnabled: Actions.OperationDisabled = {
    RETRY: false,
    RESUBMIT: false,
    SUSPEND: false,
    RESUME: false,
    STOP: false,
    TERMINATE: false,
    DELETE: false
};

const storage = new ScopedLocalStorage('ListOptions');

export function WorkflowsList({match, location, history}: RouteComponentProps<any>) {
    const queryParams = new URLSearchParams(location.search);
    const {navigation} = useContext(Context);

    const [namespace, setNamespace] = useState(Utils.getNamespace(match.params.namespace) || '');
    const [pagination, setPagination] = useState<Pagination>(() => {
        const savedPaginationLimit = storage.getItem('options', {}).paginationLimit || 0;
        return {
            offset: queryParams.get('name'),
            limit: parseLimit(queryParams.get('limit')) || savedPaginationLimit || 50
        };
    });
    const [selectedPhases, setSelectedPhases] = useState<WorkflowPhase[]>(() => {
        const savedPhases = storage.getItem('options', {}).selectedPhases || [];
        const phaseQueryParam = queryParams.getAll('phase') as WorkflowPhase[];
        return phaseQueryParam.length > 0 ? phaseQueryParam : savedPhases;
    });
    const [selectedLabels, setSelectedLabels] = useState<string[]>(() => {
        const savedLabels = storage.getItem('options', {}).selectedLabels || [];
        const labelQueryParam = queryParams.getAll('label');
        return labelQueryParam.length > 0 ? labelQueryParam : savedLabels;
    });
    const [createdAfter, setCreatedAfter] = useState<Date>();
    const [finishedBefore, setFinishedBefore] = useState<Date>();
    const [selectedWorkflows, setSelectedWorkflows] = useState(new Map<string, models.Workflow>());
    const [batchActionDisabled, setBatchActionDisabled] = useState({...allBatchActionsEnabled});
    const [workflows, setWorkflows] = useState<Workflow[]>();
    const [links, setLinks] = useState<models.Link[]>([]);
    const [columns, setColumns] = useState<models.Column[]>([]);
    const [error, setError] = useState<Error>();

    const listWatch = useRef<ListWatch<Workflow>>(null);

    function getSidePanel() {
        return queryParams.get('sidePanel');
    }

    function updateCurrentlySelectedAndBatchActions(newSelectedWorkflows: Map<string, Workflow>): void {
        const actions: any = Actions.WorkflowOperationsMap;
        const nowDisabled: any = {...allBatchActionsEnabled};
        for (const action of Object.keys(nowDisabled)) {
            for (const wf of Array.from(newSelectedWorkflows.values())) {
                nowDisabled[action] = nowDisabled[action] || actions[action].disabled(wf);
            }
        }
        setBatchActionDisabled(nowDisabled);
        setSelectedWorkflows(new Map<string, models.Workflow>(newSelectedWorkflows));
    }

    function saveHistory(namespace: string, selectedPhases: WorkflowPhase[], selectedLabels: string[], pagination: Pagination) {
        const options: WorkflowListRenderOptions = {} as WorkflowListRenderOptions;
        options.selectedPhases = selectedPhases;
        options.selectedLabels = selectedLabels;
        if (pagination.limit) {
            options.paginationLimit = pagination.limit;
        }
        storage.setItem('options', options, {} as WorkflowListRenderOptions);

        const params = new URLSearchParams();
        selectedPhases?.forEach(phase => params.append('phase', phase));
        selectedLabels?.forEach(label => params.append('label', label));
        if (pagination.offset) {
            params.append('offset', pagination.offset);
        }
        if (pagination.limit) {
            params.append('limit', pagination.limit.toString());
        }
        history.push(historyUrl('workflows' + (Utils.managedNamespace ? '' : '/{namespace}'), {namespace, extraSearchParams: params}));
    }

    function fetchWorkflows(namespace: string, selectedPhases: WorkflowPhase[], selectedLabels: string[], createdAfter: Date, finishedBefore: Date, pagination: Pagination): void {
        listWatch.current?.stop();
        listWatch.current = new ListWatch(
            () =>
                services.workflows.list(namespace, selectedPhases, selectedLabels, pagination).then(x => {
                    x.items = x.items?.filter(w => nullSafeTimeFilter(createdAfter, finishedBefore, w));
                    return x;
                }),
            (resourceVersion: string) => services.workflows.watchFields({namespace, phases: selectedPhases, labels: selectedLabels, resourceVersion}),
            metadata => {
                setError(null);
                setNamespace(namespace);
                setPagination({offset: pagination.offset, limit: pagination.limit, nextOffset: metadata.continue});
                setSelectedPhases(selectedPhases);
                setSelectedLabels(selectedLabels);
                setCreatedAfter(createdAfter);
                setFinishedBefore(finishedBefore);
                setSelectedWorkflows(new Map<string, models.Workflow>());
                saveHistory(namespace, selectedPhases, selectedLabels, pagination);
            },
            () => setError(null),
            workflows => setWorkflows(workflows.slice(0, pagination.limit || 999999)),
            err => setError(err),
            sortByYouth
        );
        listWatch.current.start();
    }

    function changeFilters(namespace: string, selectedPhases: WorkflowPhase[], selectedLabels: string[], createdAfter: Date, finishedBefore: Date, pagination: Pagination) {
        fetchWorkflows(namespace, selectedPhases, selectedLabels, createdAfter, finishedBefore, pagination);
    }

    useEffect(() => {
        (async () => {
            const info = await services.info.getInfo();
            setLinks((info.links || []).filter(link => link.scope === 'workflow-list'));
            setColumns(info.columns);
        })();
    }, []);

    useEffect(() => {
        fetchWorkflows(namespace, selectedPhases, selectedLabels, createdAfter, finishedBefore, pagination);

        return () => {
            setSelectedWorkflows(new Map<string, models.Workflow>());
            listWatch.current?.stop();
        };
    }, []);

    useCollectEvent('openedWorkflowList');

    function renderWorkflows() {
        const counts = countsByCompleted(workflows);
        return (
            <>
                <ErrorNotice error={error} />
                {!workflows ? (
                    <Loading />
                ) : workflows.length === 0 ? (
                    <ZeroState title='No workflows'>
                        <p>To create a new workflow, use the button above.</p>
                        <p>
                            <ExampleManifests />.
                        </p>
                    </ZeroState>
                ) : (
                    <>
                        {(counts.complete > 100 || counts.incomplete > 100) && (
                            <CostOptimisationNudge name='workflow-list'>
                                You have at least {counts.incomplete} incomplete, and {counts.complete} complete workflows. Reducing these amounts will reduce your costs.
                            </CostOptimisationNudge>
                        )}
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
                                                setSelectedLabels(newTags);
                                            }
                                            changeFilters(namespace, selectedPhases, newTags, createdAfter, finishedBefore, pagination);
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
                        <PaginationPanel
                            onChange={pagination => changeFilters(namespace, selectedPhases, selectedLabels, createdAfter, finishedBefore, pagination)}
                            pagination={pagination}
                            numRecords={(workflows || []).length}
                        />
                    </>
                )}
            </>
        );
    }

    return (
        <Page
            title='Workflows'
            toolbar={{
                breadcrumbs: [
                    {title: 'Workflows', path: uiUrl('workflows')},
                    {title: namespace, path: uiUrl('workflows/' + namespace)}
                ],
                actionMenu: {
                    items: [
                        {
                            title: 'Submit New Workflow',
                            iconClassName: 'fa fa-plus',
                            action: () => navigation.goto('.', {sidePanel: 'submit-new-workflow'})
                        },
                        ...links.map(link => ({
                            title: link.name,
                            iconClassName: 'fa fa-external-link',
                            action: () => (window.location.href = link.url)
                        }))
                    ]
                }
            }}>
            <WorkflowsToolbar
                selectedWorkflows={selectedWorkflows}
                clearSelection={() => setSelectedWorkflows(new Map<string, models.Workflow>())}
                loadWorkflows={() => {
                    setSelectedWorkflows(new Map<string, models.Workflow>());
                    changeFilters(namespace, selectedPhases, selectedLabels, createdAfter, finishedBefore, {
                        limit: pagination.limit
                    });
                }}
                isDisabled={batchActionDisabled}
            />
            <div className={`row ${selectedWorkflows.size === 0 ? '' : 'pt-60'}`}>
                <div className='columns small-12 xlarge-2'>
                    <WorkflowsSummaryContainer workflows={workflows} />
                    <div>
                        <WorkflowFilters
                            workflows={workflows || []}
                            namespace={namespace}
                            phaseItems={WorkflowPhases}
                            selectedPhases={selectedPhases}
                            selectedLabels={selectedLabels}
                            createdAfter={createdAfter}
                            finishedBefore={finishedBefore}
                            onChange={(namespace, selectedPhases, selectedLabels, createdAfter, finishedBefore) =>
                                changeFilters(namespace, selectedPhases, selectedLabels, createdAfter, finishedBefore, {limit: pagination.limit})
                            }
                        />
                    </div>
                </div>
                <div className='columns small-12 xlarge-10'>{renderWorkflows()}</div>
            </div>
            <SlidingPanel isShown={!!getSidePanel()} onClose={() => navigation.goto('.', {sidePanel: null})}>
                {getSidePanel() === 'submit-new-workflow' && (
                    <WorkflowCreator
                        namespace={Utils.getNamespaceWithDefault(namespace)}
                        onCreate={wf => navigation.goto(uiUrl(`workflows/${wf.metadata.namespace}/${wf.metadata.name}`))}
                    />
                )}
            </SlidingPanel>
        </Page>
    );
}

function nullSafeTimeFilter(createdAfter: Date, finishedBefore: Date, w: Workflow): boolean {
    const createdAt = w.metadata.creationTimestamp; // this should always be defined
    const finishedAt = w.status.finishedAt; // this can be undefined
    const createdDate: Date = new Date(createdAt);
    const finishedDate: Date = new Date(finishedAt);

    // check for undefined date filters as well
    // equivalent to back-end logic: https://github.com/argoproj/argo-workflows/blob/f5e31f8f36b32883087f783cb1227490bbe36bbd/pkg/apis/workflow/v1alpha1/workflow_types.go#L222
    if (createdAfter && finishedBefore) {
        return createdDate > createdAfter && finishedAt && finishedDate < finishedBefore;
    } else if (createdAfter && !finishedBefore) {
        return createdDate > createdAfter;
    } else if (!createdAfter && finishedBefore) {
        return finishedAt && finishedDate < finishedBefore;
    } else {
        return true;
    }
}

function countsByCompleted(workflows?: Workflow[]) {
    const counts = {complete: 0, incomplete: 0};
    (workflows || []).forEach(wf => {
        if (wf.metadata?.labels && wf.metadata?.labels[labels.completed] === 'true') {
            counts.complete++;
        } else {
            counts.incomplete++;
        }
    });
    return counts;
}
