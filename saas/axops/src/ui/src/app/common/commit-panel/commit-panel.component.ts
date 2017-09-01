import { Component, Input, Output, OnInit, ViewChild, EventEmitter } from '@angular/core';

import { Commit, CommitJob } from '../../model';
import { DropDownComponent } from 'argo-ui-lib/src/components';
import { PieChartInput } from '../';

@Component({
    selector: 'ax-commit-panel',
    templateUrl: './commit-panel.html',
    styles: [ require('./commit-panel.scss') ],
})
export class CommitPanelComponent implements OnInit {

    public localCommit: Commit = new Commit();
    public userName: string;
    public userEmail: string;

    private chartData: PieChartInput[] = [];

    @ViewChild('commitAction')
    public commitAction: DropDownComponent;

    @Input()
    public isExpanded: string;

    @Input()
    public isStatic: boolean = false;

    @Input()
    public set commit(value: Commit) {
        if (value) {
            this.chartData = [];
            this.chartData.push({label: 'Successful jobs', value: value.jobs_success, color: '#18BE94'});
            this.chartData.push({label: 'Run jobs', value: value.jobs_run, color: '#48A0FF'});
            this.chartData.push({label: 'Queued jobs', value: value.jobs_wait + value.jobs_init, color: '#8FA4B1'});
            this.chartData.push({label: 'Failed jobs', value: value.jobs_fail, color: '#F00052'});
            if (value.author) {
                this.userName = value.author.substring(0, value.author.lastIndexOf('<') - 1);
                this.userEmail = value.author.substring(value.author.lastIndexOf('<') + 1, value.author.trim().length - 1);
            }
        }
        this.localCommit = value;
    };

    @Output()
    public onPlusAction: EventEmitter<any> = new EventEmitter();

    @Output()
    public onSelectCommit: EventEmitter<any> = new EventEmitter();

    public displayedJobs: CommitJob[] = [];

    ngOnInit() {
        if (this.localCommit.hasOwnProperty('jobs')) {
            this.getLastThreeJobs();
        }
    }

    getLastThreeJobs() {
        this.displayedJobs = this.localCommit.jobs.reverse().slice(0, 3);
    }

    openServiceTemplatePanel(e) {
        e.stopPropagation();
        this.onPlusAction.emit(this.localCommit);
    }

    getFailedPercentValue() {
        return (100 / (this.localCommit.jobs_fail + this.localCommit.jobs_success) * this.localCommit.jobs_fail).toFixed();
    }

    selectCommit() {
        this.onSelectCommit.next({revision: this.localCommit.revision, repo: this.localCommit.repo});
    }
}
