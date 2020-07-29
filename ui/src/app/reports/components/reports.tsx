import {Page} from 'argo-ui/src/index';
import {ChartOptions} from 'chart.js';
import 'chartjs-plugin-annotation';
import * as React from 'react';
import {Bar, ChartData} from 'react-chartjs-2';
import {RouteComponentProps} from 'react-router-dom';
import {getColorForNodePhase, NODE_PHASE, Workflow} from '../../../models';
import {uiUrl} from '../../shared/base';
import {BasePage} from '../../shared/components/base-page';
import {ErrorPanel} from '../../shared/components/error-panel';
import {InputFilter} from '../../shared/components/input-filter';
import {Loading} from '../../shared/components/loading';
import {TagsInput} from '../../shared/components/tags-input/tags-input';
import {ZeroState} from '../../shared/components/zero-state';
import {Consumer, ContextApis} from '../../shared/context';
import {denominator} from '../../shared/duration';
import {services} from '../../shared/services';

interface Chart {
    data: ChartData<any>;
    options: ChartOptions;
}

interface State {
    error?: Error;
    charts?: Chart[];
}

export class Reports extends BasePage<RouteComponentProps<any>, State> {
    private get namespace() {
        return this.props.match.params.namespace || '';
    }

    private set namespace(namespace: string) {
        document.location.href = uiUrl('reports/' + namespace);
    }

    private get labels() {
        return (this.queryParam('labels') || '').split(',').filter(i => i !== '');
    }

    private set labels(labels) {
        this.setQueryParams({labels: labels.join(',')});
    }

    private get phase() {
        return (this.labels.find(label => label.startsWith('workflows.argoproj.io/phase')) || '').replace(/workflows.argoproj.io\/phase=/, '');
    }

    private set phase(phase: string) {
        this.labels = this.labels.filter(label => !label.startsWith('workflows.argoproj.io/phase')).concat('workflows.argoproj.io/phase=' + phase);
    }

    private get canRunReport() {
        return this.namespace !== '' && this.labels.length > 0;
    }

    constructor(props: any) {
        super(props);
        this.state = {};
    }

    public componentDidMount() {
        if (!this.canRunReport) {
            return;
        }
        this.setState({charts: null}, () => {
            services.workflows
                .list(this.namespace, [], this.labels, {})
                .then(list => this.getExtractDatasets(list.items || []))
                .then(charts => this.setState({charts, error: null}))
                .catch(error => this.setState({error}));
        });
    }

    public render() {
        return (
            <Consumer>
                {ctx => (
                    <Page
                        title='Reports'
                        toolbar={{
                            breadcrumbs: [{title: 'Reports', path: uiUrl('/reports')}],
                            actionMenu: {
                                items: [
                                    {
                                        title: 'Workflow List',
                                        iconClassName: 'fa fa-stream',
                                        disabled: !this.canRunReport,
                                        action: () => ctx.navigation.goto(uiUrl(`workflows/${this.namespace}?labels=${this.labels.join(',')}`))
                                    }
                                ]
                            }
                        }}>
                        {this.renderFilters()}
                        {this.renderReport(ctx)}
                    </Page>
                )}
            </Consumer>
        );
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
            .slice(0, 100)
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
                        text: 'Resources'
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

    // @ts-ignore
    private label(label: string, checked: boolean) {
        const labels = this.labels;
        const i = labels.indexOf(label);
        if (checked && i < 0) {
            this.labels = labels.concat(label);
        } else if (!checked && i >= 0) {
            this.labels = labels.slice(i, i + 1);
        }
    }

    private renderFilters() {
        return (
            <div className='row' style={{marginTop: 25, marginBottom: 25}}>
                <div className='columns small-12'>
                    <div className='white-box'>
                        <div className='row'>
                            <div className='columns small-3' key='namespace'>
                                <InputFilter name='namespace' value={this.namespace} placeholder='Namespace' onChange={namespace => (this.namespace = namespace)} />
                            </div>
                            <div className='columns small-5' key='labels'>
                                <TagsInput placeholder='Labels' tags={this.labels} onChange={labels => (this.labels = labels)} />
                            </div>
                            <div className='columns small-4' key='phases'>
                                <p>
                                    {[NODE_PHASE.SUCCEEDED, NODE_PHASE.ERROR, NODE_PHASE.FAILED].map(phase => (
                                        <label key={phase} className='argo-button argo-button--base-o' style={{marginRight: 5}} onClick={() => (this.phase = phase)}>
                                            {this.phase === phase ? <i className='fa fa-check' /> : <i className='fa fa-times' />} {phase}
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
                        <b>cron workflows</b>, your workflows will be automatically labelled.
                    </p>
                    <p>Select a namespace and at least one label to get a report.</p>
                    <p>
                        {' '}
                        <a href='https://github.com/argoproj/argo/blob/master/docs/cost-optimisation.md'>Learn more about cost optimization</a>
                    </p>
                </ZeroState>
            );
        }
        if (this.state.error) {
            return <ErrorPanel error={this.state.error} />;
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
                                        ctx.navigation.goto(uiUrl('workflows/' + this.namespace + '/' + workflowName));
                                    }}
                                />
                            </div>
                        </div>
                    </div>
                ))}
                <div className='row' key='info'>
                    <div className='columns small-12'>
                        <small>
                            <i className='fa fa-info-circle' /> Showing {this.state.charts[0].data.labels.length} workflows. Deleted workflows (even if archived) are not shown.
                        </small>
                    </div>
                </div>
            </>
        );
    }
}
