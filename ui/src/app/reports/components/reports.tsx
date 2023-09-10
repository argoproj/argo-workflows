import {Page} from 'argo-ui/src/index';
import {ChartOptions} from 'chart.js';
import 'chartjs-plugin-annotation';
import * as React from 'react';
import {useContext, useEffect, useState} from 'react';
import {Bar, ChartData} from 'react-chartjs-2';
import {RouteComponentProps} from 'react-router-dom';
import {getColorForNodePhase, NODE_PHASE, Workflow} from '../../../models';
import {uiUrl} from '../../shared/base';
import {DataLoaderDropdown} from '../../shared/components/data-loader-dropdown';
import {ErrorNotice} from '../../shared/components/error-notice';
import {InfoIcon} from '../../shared/components/fa-icons';
import {NamespaceFilter} from '../../shared/components/namespace-filter';
import {TagsInput} from '../../shared/components/tags-input/tags-input';
import {useCollectEvent} from '../../shared/components/use-collect-event';
import {ZeroState} from '../../shared/components/zero-state';
import {Context} from '../../shared/context';
import {denominator} from '../../shared/duration';
import {Footnote} from '../../shared/footnote';
import {historyUrl} from '../../shared/history';
import {services} from '../../shared/services';
import {Utils} from '../../shared/utils';

interface Chart {
    data: ChartData<any>;
    options: ChartOptions;
}

const limit = 100;
const labelKeyPhase = 'workflows.argoproj.io/phase';
const labelKeyWorkflowTemplate = 'workflows.argoproj.io/workflow-template';
const labelKeyCronWorkflow = 'workflows.argoproj.io/cron-workflow';

export function Reports ({match, location, history}: RouteComponentProps<any>) {
    const queryParams = new URLSearchParams(location.search);
    const {navigation} = useContext(Context);

    // state for URL, query, and label parameters
    const [namespace, setNamespace] = useState<string>(Utils.getNamespace(match.params.namespace) || '');
    const [labels, setLabels] = useState((queryParams.get('labels') || '').split(',').filter(v => v !== ''));
    const [autocompleteLabels, setAutocompleteLabels] = useState(['']);
    const [charts, setCharts] = useState<Chart[]>();
    const [error, setError] = useState<Error>();

    // save history
    useEffect(() => {
        history.push(historyUrl('reports' + (Utils.managedNamespace ? '' : '/{namespace}'), {namespace, labels: labels.join(',')}))
    }, [namespace, labels]);

    async function fetchReport(newNamespace: string, newLabels: string[]) {
        if (newNamespace === '' || newLabels.length === 0) {
            setNamespace(newNamespace);
            setLabels(newLabels);
            setCharts(null);
            return;
        }

        try {
            const list = await services.workflows.list(newNamespace, [], newLabels, {limit}, [
                'items.metadata.name',
                'items.status.phase',
                'items.status.startedAt',
                'items.status.finishedAt',
                'items.status.resourcesDuration'
            ]);
            const newCharts = getExtractDatasets(list.items || []);
            setNamespace(newNamespace);
            setLabels(newLabels);
            setCharts(newCharts);
            setError(null);
        } catch (newError) {
            setError(newError);
        }
    }

    function getLabel(name: string) {
        return (labels.find(label => label.startsWith(name)) || '').replace(name + '=', '');
    }

    function setLabel(name: string, value: string) {
        fetchReport(namespace, labels.filter(label => !label.startsWith(name)).concat(name + '=' + value));
    }

    function getPhase() {
        return getLabel(labelKeyPhase);
    }

    function setPhase(value: string) {
        setLabel(labelKeyPhase, value);
    }

    function setWorkflowTemplate(value: string) {
        setLabel(labelKeyWorkflowTemplate, value);
    }

    function setCronWorkflow(value: string) {
        setLabel(labelKeyCronWorkflow, value);
    }

    useEffect(() => {
        fetchReport(namespace, labels);
    }, [namespace, labels]);

    useCollectEvent('openedReports');

    return (
        <Page
            title='Reports'
            toolbar={{
                breadcrumbs: [
                    {title: 'Reports', path: uiUrl('reports')},
                    {title: this.state.namespace, path: uiUrl('reports/' + this.state.namespace)}
                ]
            }}>
            <div className='row'>
                <div className='columns small-12 xlarge-2'>{renderFilters()}</div>

                <div className='columns small-12 xlarge-10'>
                    <ErrorNotice error={error} />;
                    {!charts ? (
                        <ZeroState title='Workflow Report'>
                            <p>
                                Use this page to find costly or time consuming workflows. You must label workflows you want to report on. If you use <b>workflow templates</b> or{' '}
                                <b>cron workflows</b>, your workflows will be automatically labelled. You'll probably need to enable the{' '}
                                <a href='https://argoproj.github.io/argo-workflows/workflow-archive/'>workflow archive</a> to get long term data. Only the {limit} most recent workflows are
                                shown.
                            </p>
                            <p>Select a namespace and at least one label to get a report.</p>
                            <p>
                                <a href='https://argoproj.github.io/argo-workflows/cost-optimisation/'>Learn more about cost optimization</a>
                            </p>
                        </ZeroState>
                    ) : (
                        <>
                            {charts.map(chart => (
                                <div key={chart.data.name}>
                                    <div className='white-box'>
                                        <Bar
                                            data={chart.data}
                                            options={chart.options}
                                            onElementsClick={(e: any[]) => {
                                                const activePoint = e[0];
                                                if (activePoint === undefined) {
                                                    return;
                                                }
                                                const workflowName = chart.data.labels[activePoint._index];
                                                navigation.goto(uiUrl('workflows/' + namespace + '/' + workflowName));
                                            }}
                                        />
                                    </div>
                                </div>
                            ))}
                            <Footnote>
                                <InfoIcon /> {charts[0].data.labels.length} records.
                            </Footnote>
                        </>
                    )}
                </div>
            </div>
        </Page>
    );

    function renderFilters() {
        return (
            <div className='wf-filters-container'>
                <div className='row'>
                    <div className=' columns small-12 xlarge-12'>
                        <p className='wf-filters-container__title'>Namespace</p>
                        <NamespaceFilter
                            value={namespace}
                            onChange={newNamespace => {
                                fetchReport(newNamespace, labels);
                            }}
                        />
                    </div>
                    <div className=' columns small-12 xlarge-12'>
                        <p className='wf-filters-container__title'>Labels</p>
                        <TagsInput
                            placeholder='Labels'
                            tags={labels}
                            autocomplete={autocompleteLabels}
                            onChange={newLabels => fetchReport(namespace, newLabels)}
                        />
                    </div>
                    <div className=' columns small-12 xlarge-12'>
                        <p className='wf-filters-container__title'>Workflow Template</p>
                        <DataLoaderDropdown
                            load={() =>
                                services.workflowTemplate
                                    .list(namespace, [])
                                    .then(list => list.items || [])
                                    .then(list => list.map(x => x.metadata.name))
                            }
                            onChange={value => (setWorkflowTemplate(value))}
                        />
                    </div>
                    <div className=' columns small-12 xlarge-12'>
                        <p className='wf-filters-container__title'>Cron Workflow</p>
                        <DataLoaderDropdown
                            load={() => services.cronWorkflows.list(namespace).then(list => list.map(x => x.metadata.name))}
                            onChange={value => (setCronWorkflow(value))}
                        />
                    </div>
                    <div className=' columns small-12 xlarge-12'>
                        <p className='wf-filters-container__title'>Phase</p>
                        {[NODE_PHASE.SUCCEEDED, NODE_PHASE.ERROR, NODE_PHASE.FAILED].map(phase => (
                            <div key={phase}>
                                <label style={{marginRight: 10}}>
                                    <input type='radio' checked={phase === getPhase()} onChange={() => (setPhase(phase))} style={{marginRight: 5}} />
                                    {phase}
                                </label>
                            </div>
                        ))}
                    </div>
                </div>
            </div>
        );
    }
}

// pure function on the workflows (no requests, state, props, context, etc)
function getExtractDatasets(workflows: Workflow[]) {
    const filteredWorkflows = workflows
        .filter(wf => !!wf.status.finishedAt)
        .map(wf => ({
            name: wf.metadata.name,
            finishedAt: new Date(wf.status.finishedAt),
            startedAt: new Date(wf.status.startedAt),
            phase: wf.status.phase,
            resourcesDuration: wf.status.resourcesDuration
        }))
        .sort((a, b) => b.finishedAt.getTime() - a.finishedAt.getTime())
        .slice(0, limit)
        .reverse();

    const labels: string[] = new Array(filteredWorkflows.length);
    const backgroundColors: string[] = new Array(filteredWorkflows.length);
    const durationData: number[] = new Array(filteredWorkflows.length);
    const resourceData = {} as any;

    filteredWorkflows.forEach((wf, i) => {
        labels[i] = wf.name;
        backgroundColors[i] = getColorForNodePhase(wf.phase);
        durationData[i] = (wf.finishedAt.getTime() - wf.startedAt.getTime()) / 1000;
        Object.entries(wf.resourcesDuration || {}).forEach(([resource, value]) => {
            if (!resourceData[resource]) {
                resourceData[resource] = new Array(filteredWorkflows.length);
            }
            resourceData[resource][i] = value;
        });
    });
    const resourceColors = {
        'cpu': 'teal',
        'memory': 'blue',
        'storage': 'purple',
        'ephemeral-storage': 'purple'
    } as any;

    return [
        {
            data: {
                name: 'duration',
                labels,
                datasets: [
                    {
                        data: durationData,
                        backgroundColor: backgroundColors
                    }
                ]
            },
            options: {
                title: {
                    display: true,
                    text: 'Duration'
                },
                legend: {display: false},
                scales: {
                    xAxes: [{}],
                    yAxes: [
                        {
                            id: 'duration',
                            ticks: {
                                beginAtZero: true
                            },
                            scaleLabel: {
                                display: true,
                                labelString: 'Duration (seconds)'
                            }
                        }
                    ]
                },
                annotation: {
                    annotations: [
                        {
                            type: 'line',
                            mode: 'horizontal',
                            scaleID: 'duration',
                            value: durationData.length > 0 ? durationData.reduce((a, b) => a + b, 0) / durationData.length : 0,
                            borderColor: 'gray',
                            borderWidth: 1,
                            label: {
                                enabled: true,
                                position: 'left',
                                content: 'Average'
                            }
                        }
                    ]
                }
            }
        },
        {
            data: {
                name: 'resources',
                labels,
                datasets: Object.entries(resourceData).map(([resource, data]) => ({
                    yAxesID: resource,
                    label: resource,
                    data,
                    backgroundColor: resourceColors[resource] || 'black'
                }))
            },
            options: {
                title: {
                    display: true,
                    text: 'Resources (not available for archived workflows)'
                },
                scales: {
                    xAxes: [{}],
                    yAxes: Object.keys(resourceData).map(resource => ({
                        id: resource,
                        ticks: {
                            beginAtZero: true
                        },
                        scaleLabel: {
                            display: true,
                            labelString: resource + ' (' + denominator(resource) + ')'
                        }
                    }))
                }
            }
        }
    ];
}
