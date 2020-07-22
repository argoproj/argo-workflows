import {Page} from 'argo-ui/src/index';
import {ChartScales} from 'chart.js';
import * as React from 'react';
import {ChartData, Line} from 'react-chartjs-2';
import {RouteComponentProps} from 'react-router-dom';
import {getColorForNodePhase, Workflow} from '../../../models';
import {uiUrl} from '../../shared/base';
import {BasePage} from '../../shared/components/base-page';
import {ErrorPanel} from '../../shared/components/error-panel';
import {InputFilter} from '../../shared/components/input-filter';
import {Loading} from '../../shared/components/loading';
import {TagsInput} from '../../shared/components/tags-input/tags-input';
import {ZeroState} from '../../shared/components/zero-state';
import {Consumer} from '../../shared/context';
import {denominator} from '../../shared/duration';
import {services} from '../../shared/services';

interface State {
    error?: Error;
    data?: ChartData<any>;
    scales?: ChartScales;
    numWorkflows?: number;
}

const maxAge = 90;

export class Reports extends BasePage<RouteComponentProps<any>, State> {
    constructor(props: any) {
        super(props);
        this.state = {};
    }

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

    private get ready() {
        return this.namespace !== '' && this.labels.length > 0;
    }

    public componentDidMount() {
        if (!this.ready) {
            return;
        }

        const extractDatasets = (workflows: Workflow[]): {scales: Chart.ChartScales; datasets: any[]; numWorkflows: number} => {
            const datasets = new Map<string, any>();
            const scales: ChartScales = {
                xAxes: [
                    {
                        type: 'time',
                        scaleLabel: {
                            labelString: 'Finished At'
                        }
                    }
                ],
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
            };
            const thirtyDaysAgo = new Date().getTime() - maxAge * 24 * 60 * 60 * 1000;

            const filteredWorkflows = workflows.filter(wf => wf.status.finishedAt !== '' && new Date(wf.status.finishedAt).getTime() > thirtyDaysAgo);
            filteredWorkflows.forEach(wf => {
                const finishedAt = new Date(wf.status.finishedAt);
                const startedAt = new Date(wf.status.startedAt);
                const duration: number = finishedAt.getTime() - startedAt.getTime();

                const phase = wf.status.phase;

                if (!datasets.has(phase)) {
                    datasets.set(phase, {
                        yAxisID: 'duration',
                        data: [],
                        fill: false,
                        borderColor: getColorForNodePhase(phase),
                        lineTension: 0,
                        label: phase
                    });
                }

                datasets.get(phase).data.push({t: finishedAt, y: duration / 1000, label: wf.metadata.name});

                Object.keys(wf.status.resourcesDuration).forEach(resource => {
                    if (!datasets.has(resource)) {
                        datasets.set(resource, {
                            yAxisID: resource,
                            data: [],
                            fill: false,
                            borderColor: '#000',
                            lineTension: 0,
                            label: resource
                        });
                        scales.yAxes.push({
                            id: resource,
                            ticks: {
                                beginAtZero: true
                            },
                            position: 'right',
                            scaleLabel: {
                                display: true,
                                labelString: resource + ' (' + denominator(resource) + ')'
                            }
                        });
                    }
                    datasets.get(resource).data.push({
                        t: finishedAt,
                        y: wf.status.resourcesDuration[resource],
                        label: wf.metadata.name
                    });
                });
            });
            return {datasets: Array.from(datasets.values()), scales, numWorkflows: filteredWorkflows.length};
        };

        services.workflows
            .list(this.namespace, [], this.labels, {})
            .then(list => extractDatasets(list.items || []))
            .then(item =>
                this.setState({
                    numWorkflows: item.numWorkflows,
                    data: {
                        datasets: item.datasets
                    },
                    scales: item.scales,
                    error: null
                })
            )
            .catch(error => this.setState({error}));
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
                                        disabled: !this.ready,
                                        action: () => ctx.navigation.goto(uiUrl(`workflows/${this.namespace}?labels=${this.labels.join(',')}`))
                                    }
                                ]
                            }
                        }}>
                        {this.renderFilters()}
                        {this.renderReport()}
                    </Page>
                )}
            </Consumer>
        );
    }

    private renderFilters() {
        return (
            <div className='row' style={{marginTop: 25, marginBottom: 25}}>
                <div className='columns small-12'>
                    <div className='white-box'>
                        <div className='row'>
                            <div className='columns small-3'>
                                <InputFilter name='namespace' value={this.namespace} placeholder='Namespace' onChange={namespace => (this.namespace = namespace)} />
                            </div>
                            <div className='columns small-3'>
                                <TagsInput placeholder='Labels' tags={this.labels} onChange={labels => (this.labels = labels)} />
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        );
    }

    private renderReport() {
        if (!this.ready) {
            return (
                <ZeroState title='Workflow Report'>
                    <p>
                        Use this page to find costly or time consuming workflows. You must label workflows you want to report on. If you use <b>workflow templates</b> or{' '}
                        <b>cron workflows</b>, your workflows will be automatacially labelled.
                    </p>
                    <p>Select a namespace and at least one label to get a report.</p>
                </ZeroState>
            );
        }
        if (this.state.error) {
            return <ErrorPanel error={this.state.error} />;
        }
        if (!this.state.data) {
            return <Loading />;
        }
        return (
            <Consumer>
                {ctx => (
                    <div className='row'>
                        <div className='columns small-12'>
                            <div className='white-box'>
                                <Line
                                    data={this.state.data}
                                    options={{
                                        scales: this.state.scales
                                    }}
                                    onElementsClick={(e: any[]) => {
                                        const activePoint = e[0];
                                        if (activePoint === undefined) {
                                            return;
                                        }
                                        const workflowName = this.state.data.datasets[activePoint._datasetIndex].data[activePoint._index].label;
                                        ctx.navigation.goto(uiUrl('workflows/' + this.namespace + '/' + workflowName));
                                    }}
                                />
                            </div>
                            <small>
                                <i className='fa fa-info-circle' /> Showing {this.state.numWorkflows} workflows finished in the last {maxAge} days. Deleted workflows (even if
                                archived) are not shown.
                            </small>
                        </div>
                    </div>
                )}
            </Consumer>
        );
    }
}
