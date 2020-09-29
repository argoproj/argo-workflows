import {Checkbox, Page} from 'argo-ui/src/index';
import {ChartOptions} from 'chart.js';
import 'chartjs-plugin-annotation';
import * as React from 'react';
import {Bar, ChartData} from 'react-chartjs-2';
import {RouteComponentProps} from 'react-router-dom';
import {getColorForNodePhase, NODE_PHASE, Workflow} from '../../../models';
import {uiUrl} from '../../shared/base';
import {BasePage} from '../../shared/components/base-page';
import {ErrorNotice} from '../../shared/components/error-notice';
import {InputFilter} from '../../shared/components/input-filter';
import {Loading} from '../../shared/components/loading';
import {TagsInput} from '../../shared/components/tags-input/tags-input';
import {ZeroState} from '../../shared/components/zero-state';
import {Consumer, ContextApis} from '../../shared/context';
import {denominator} from '../../shared/duration';
import {services} from '../../shared/services';
import {Utils} from '../../shared/utils';

interface Chart {
    data: ChartData<any>;
    options: ChartOptions;
}

interface State {
    archivedWorkflows: boolean;
    namespace: string;
    labels: string[];
    error?: Error;
    charts?: Chart[];
}

const limit = 100;

export class Reports extends BasePage<RouteComponentProps<any>, State> {
    private get phase() {
        return (this.state.labels.find(label => label.startsWith('workflows.argoproj.io/phase')) || '').replace(/workflows.argoproj.io\/phase=/, '');
    }

    private set phase(phase: string) {
        this.fetchReport(
            this.state.namespace,
            this.state.labels.filter(label => !label.startsWith('workflows.argoproj.io/phase')).concat('workflows.argoproj.io/phase=' + phase),
            this.state.archivedWorkflows
        );
    }

    private get canRunReport() {
        return this.state.namespace !== '' && this.state.labels.length > 0;
    }

    constructor(props: RouteComponentProps<any>, context: any) {
        super(props, context);
        this.state = {
            archivedWorkflows: !!this.queryParam('archivedWorkflows'),
            namespace: this.props.match.params.namespace || '',
            labels: (this.queryParam('labels') || '').split(',').filter(v => v !== '')
        };
    }

    public componentDidMount() {
        this.fetchReport(this.state.namespace, this.state.labels, this.state.archivedWorkflows);
    }

    public render() {
        return (
            <Consumer>
                {ctx => (
                    <Page title='Reports' toolbar={{breadcrumbs: [{title: 'Reports', path: uiUrl('/reports')}]}}>
                        {this.renderFilters()}
                        {this.renderReport(ctx)}
                    </Page>
                )}
            </Consumer>
        );
    }

    private fetchReport(namespace: string, labels: string[], archivedWorkflows: boolean) {
        this.setState({charts: null, namespace, labels, archivedWorkflows}, () => {
            if (!this.canRunReport) {
                return;
            }
            (archivedWorkflows
                ? services.archivedWorkflows.list(namespace, [], labels, null, null, {limit})
                : services.workflows.list(namespace, [], labels, {limit}, [
                      'items.metadata.name',
                      'items.status.phase',
                      'items.status.startedAt',
                      'items.status.finishedAt',
                      'items.status.resourcesDuration'
                  ])
            )
                .then(list => this.getExtractDatasets(list.items || []))
                .then(charts => this.setState({charts, error: null}, () => this.saveHistory()))
                .catch(error => this.setState({error}));
        });
    }

    private saveHistory() {
        this.url = uiUrl('reports/' + this.state.namespace + '?archivedWorkflows=' + this.state.archivedWorkflows + '&labels=' + this.state.labels.join(','));
        Utils.setCurrentNamespace(this.state.namespace);
    }

    private getExtractDatasets(workflows: Workflow[]) {
        const filteredWorkflows = workflows
            .filter(wf => wf.status.finishedAt !== '')
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
                        text: 'Resources (Not available for archived workflows)'
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

    private renderFilters() {
        return (
            <div className='row' style={{marginTop: 25, marginBottom: 25}}>
                <div className='columns small-12'>
                    <div className='white-box'>
                        <div className='row'>
                            <div className='columns small-4' key='archived-workflows'>
                                <label>
                                    <Checkbox
                                        checked={this.state.archivedWorkflows}
                                        onChange={checked => {
                                            this.fetchReport(this.state.namespace, this.state.labels, checked);
                                        }}
                                    />{' '}
                                    Archived Workflows
                                </label>
                            </div>
                            <div className='columns small-8' key='namespace'>
                                <InputFilter
                                    name='namespace'
                                    value={this.state.namespace}
                                    placeholder='Namespace'
                                    onChange={namespace => this.fetchReport(namespace, this.state.labels, this.state.archivedWorkflows)}
                                />
                            </div>
                        </div>
                        <div className='row'>
                            <div className='columns small-8' key='labels'>
                                <TagsInput
                                    placeholder='Labels'
                                    tags={this.state.labels}
                                    onChange={labels => this.fetchReport(this.state.namespace, labels, this.state.archivedWorkflows)}
                                />
                            </div>
                            <div className='columns small-4' key='phases'>
                                <p>
                                    {[NODE_PHASE.SUCCEEDED, NODE_PHASE.ERROR, NODE_PHASE.FAILED].map(phase => (
                                        <label key={phase} style={{marginRight: 10}}>
                                            <input type='radio' checked={phase === this.phase} onChange={() => (this.phase = phase)} style={{marginRight: 5}} />
                                            {phase}
                                        </label>
                                    ))}
                                </p>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        );
    }

    private renderReport(ctx: ContextApis) {
        if (!this.canRunReport) {
            return (
                <ZeroState title='Workflow Report'>
                    <p>
                        Use this page to find costly or time consuming workflows. You must label workflows you want to report on. If you use <b>workflow templates</b> or{' '}
                        <b>cron workflows</b>, your workflows will be automatically labelled. You'll probably need to enable the{' '}
                        <a href='https://argoproj.github.io/argo/workflow-archive/'>workflow archive</a> to get long term data. Only the 100 most recent workflows are shown.
                    </p>
                    <p>Select a namespace and at least one label to get a report.</p>
                    <p>
                        <a href='https://argoproj.github.io/argo/cost-optimisation/'>Learn more about cost optimization</a>
                    </p>
                </ZeroState>
            );
        }
        if (this.state.error) {
            return <ErrorNotice error={this.state.error} style={{margin: 20}} />;
        }
        if (!this.state.charts) {
            return <Loading />;
        }
        return (
            <>
                {this.state.charts.map(chart => (
                    <div className='row' key={chart.data.name}>
                        <div className='columns small-12'>
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
                                        ctx.navigation.goto(uiUrl('workflows/' + this.state.namespace + '/' + workflowName));
                                    }}
                                />
                            </div>
                        </div>
                    </div>
                ))}
                <div className='row' key='info'>
                    <div className='columns small-12'>
                        <small>
                            <i className='fa fa-info-circle' /> Showing {this.state.charts[0].data.labels.length} workflows.
                        </small>
                    </div>
                </div>
            </>
        );
    }
}
