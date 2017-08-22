import { Component, Input } from '@angular/core';
import { FixtureInstance } from '../../../model';

@Component({
    selector: 'ax-fixture-usage-chart',
    template: `
        <div class="fixture-usage-chart">
            <nvd3 [options]="chartOptions" [data]="chartData"></nvd3>
            <p class="fixture-usage-chart__title fixture-usage-chart__title--deployments">DEPLOYMENTS: {{usageInfo.deploymentsCount}}</p>
            <p class="fixture-usage-chart__title fixture-usage-chart__title--workflows">WORKFLOWS: {{usageInfo.workflowsCount}}</p>
        </div>`,
    styles: [ require('./fixture-usage-chart.scss') ]
})
export class FixtureUsageChartComponent {
    public chartOptions = {
        chart: {
            type: 'pieChart',
            height: 120,
            margin: { top: 0, left: 0, right: 0, bottom: 0 },
            showLabels: false,
            duration: 500,
            labelThreshold: 0.01,
            labelSunbeamLayout: true,
            showLegend: false,
            donut: true,
            donutRatio: 0.54,
            tooltip: { enabled: false },
            color: [
                // deployment color
                '#1FBDD0',
                // workflow color
                '#95D58F',
                // not used
                '#F5FBFD'
            ]
        }
    };
    public usageInfo = { deploymentsCount: 0, workflowsCount: 0 };
    public chartData = [];

    @Input()
    public set fixture(fixture: FixtureInstance) {
        this.usageInfo = {
            deploymentsCount: (fixture.referrers || []).filter(ref => ref.application_id).length,
            workflowsCount: (fixture.referrers || []).filter(ref => !ref.application_id).length
        };
        if (fixture.concurrency && fixture.concurrency > 0) {
            this.chartData = [
                { y: this.usageInfo.deploymentsCount },
                { y: this.usageInfo.workflowsCount },
                { y: fixture.concurrency - (this.usageInfo.deploymentsCount + this.usageInfo.workflowsCount)
            }];
        } else {
            this.chartData = [{y: 0}, {y: 0}, {y: 1}];
        }
    }
}
