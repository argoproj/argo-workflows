import {getColorForNodePhase, Workflow} from '../../../models';
import {denominator} from '../../shared/duration';

export function workflowsToChartData(workflows: Workflow[], limit: number) {
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
