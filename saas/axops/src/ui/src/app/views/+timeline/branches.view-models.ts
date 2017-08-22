import * as moment from 'moment';
import { Task } from '../../model';

export interface JobsTimelineInput {
    startTime: moment.Moment;
    endTime: moment.Moment;
    tasks: Task[];
}

export interface BranchInfo {
    name: string;
    repo: string;
    shortcutRepoBranch: string;
    mostRecentCommitUnitTime: number;
    failedJobsCount: number;
    scheduledJobsCount: number;
    successfulJobsCount: number;
    runningJobsCount: number;
    canceledJobsCount: number;
    timelineInput: JobsTimelineInput;
}

export class JobFilter {
    failed: boolean = true;
    delayed:  boolean = true;
    succeeded: boolean = true;
    running: boolean = true;

    public get allSelected() {
        return this.failed && this.delayed && this.succeeded && this.running;
    }
}

export class NowLine {
    left: string = '';
    now: number = null;
    inRange: boolean = false;
}
