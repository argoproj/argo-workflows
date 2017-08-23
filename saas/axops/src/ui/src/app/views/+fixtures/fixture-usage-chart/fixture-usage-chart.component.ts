import { Component, Input } from '@angular/core';
import { FixtureInstance } from '../../../model';

@Component({
    selector: 'ax-fixture-usage-chart',
    templateUrl: './fixture-usage-chart.html',
    styles: [ require('./fixture-usage-chart.scss') ]
})
export class FixtureUsageChartComponent {
    public chartData = [];
    public usageInfo = { deploymentsCount: 0, workflowsCount: 0 };

    @Input()
    public set fixture(fixture: FixtureInstance) {
        this.usageInfo = {
            deploymentsCount: (fixture.referrers || []).filter(ref => ref.application_id).length,
            workflowsCount: (fixture.referrers || []).filter(ref => !ref.application_id).length
        };
        if (fixture.concurrency && fixture.concurrency > 0) {
            this.chartData = [];
            this.chartData.push({label: 'Deployments', value: this.usageInfo.deploymentsCount, color: '#bf86f1'});
            this.chartData.push({label: 'Workflows', value: this.usageInfo.workflowsCount, color: '#0055b9'});
            this.chartData.push({label: '', value: fixture.concurrency - (this.usageInfo.deploymentsCount + this.usageInfo.workflowsCount), color: '#ccd6dd'});
        } else {
            this.chartData = [{y: 0}, {y: 0}, {y: 1}];
        }
    }
}
