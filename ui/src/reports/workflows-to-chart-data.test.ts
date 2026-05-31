import {Workflow} from '../shared/models';
import {workflowsToChartData} from './workflows-to-chart-data';

function wf(name: string, startedAt: string, finishedAt: string, resourcesDuration?: {[k: string]: number}): Workflow {
    return {
        metadata: {name},
        status: {
            phase: 'Succeeded',
            startedAt,
            finishedAt,
            resourcesDuration
        }
    } as unknown as Workflow;
}

describe('workflowsToChartData', () => {
    const workflows = [
        wf('wf-a', '2020-01-01T00:00:00Z', '2020-01-01T00:00:10Z', {cpu: 5, memory: 7}),
        wf('wf-b', '2020-01-02T00:00:00Z', '2020-01-02T00:00:30Z', {cpu: 15, memory: 21})
    ];

    it('produces v4-shaped options for the duration chart', () => {
        const [durationChart] = workflowsToChartData(workflows, 100);
        const options = durationChart.options as any;

        // scales use keyed objects, not xAxes/yAxes arrays
        expect(options.scales.x).toBeDefined();
        expect((options.scales as any).xAxes).toBeUndefined();
        expect((options.scales as any).yAxes).toBeUndefined();
        expect(options.scales.duration.title.text).toBe('Duration (seconds)');
        expect(options.scales.duration.beginAtZero).toBe(true);

        // every bar dataset must declare a yAxisID that matches an existing value scale.
        // In chart.js v4 a bar dataset with no yAxisID defaults to scale 'y'; since this
        // chart renames its value scale to 'duration', a missing yAxisID throws
        // "Cannot read properties of undefined (reading 'axis')" and crashes the page.
        expect(durationChart.data.datasets.length).toBeGreaterThan(0);
        durationChart.data.datasets.forEach((ds: any) => {
            expect(ds.yAxisID).toBeDefined();
            expect(options.scales[ds.yAxisID]).toBeDefined();
            expect(ds.yAxisID).toBe('duration');
        });

        // title and legend moved under plugins
        expect(options.plugins.title.text).toBe('Duration');
        expect(options.plugins.legend.display).toBe(false);

        // annotation moved under plugins.annotation with keyed annotations
        const avgLine = options.plugins.annotation.annotations.avgLine;
        expect(avgLine.type).toBe('line');
        expect(avgLine.scaleID).toBe('duration');
        // average of 10s and 30s == 20s
        expect(avgLine.yMin).toBe(20);
        expect(avgLine.yMax).toBe(20);
        expect(avgLine.label.display).toBe(true);
        expect(avgLine.label.position).toBe('start');
        expect(avgLine.label.content).toBe('Average');
    });

    it('produces v4-shaped options + keyed scales for the resources chart', () => {
        const [, resourcesChart] = workflowsToChartData(workflows, 100);
        const options = resourcesChart.options as any;

        expect(options.scales.x).toBeDefined();
        expect(options.scales.cpu.title.text).toContain('cpu');
        expect(options.scales.memory.title.text).toContain('memory');
        expect(options.scales.cpu.beginAtZero).toBe(true);
        expect(options.plugins.title.text).toContain('Resources');

        // datasets use the v4 `yAxisID` (not the v2 `yAxesID` typo); every bar dataset
        // must declare a yAxisID that matches an existing value scale, or chart.js v4
        // throws "Cannot read properties of undefined (reading 'axis')".
        expect(resourcesChart.data.datasets.length).toBeGreaterThan(0);
        resourcesChart.data.datasets.forEach((ds: any) => {
            expect(ds.yAxisID).toBeDefined();
            expect(ds.yAxesID).toBeUndefined();
            expect(options.scales[ds.yAxisID]).toBeDefined();
        });
    });

    it('keeps the non-standard name field used as a React key', () => {
        const [durationChart, resourcesChart] = workflowsToChartData(workflows, 100);
        expect(durationChart.data.name).toBe('duration');
        expect(resourcesChart.data.name).toBe('resources');
    });
});
