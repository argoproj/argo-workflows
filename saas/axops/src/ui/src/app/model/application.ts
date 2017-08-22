import { Deployment, ExternalRoute } from '.';

export const APPLICATION_STATUSES = {
    WAITING: 'Waiting',
    ACTIVE: 'Active',
    ERROR: 'Error',
    TERMINATED: 'Terminated',
    TERMINATING: 'Terminating',
    STOPPED: 'Stopped',
    STOPPING: 'Stopping',
    INIT: 'Init',
    UPGRADING: 'Upgrading',
};

export const ACTIONS_BY_STATUS = {
    TERMINATE: [APPLICATION_STATUSES.ACTIVE, APPLICATION_STATUSES.ERROR, APPLICATION_STATUSES.WAITING, APPLICATION_STATUSES.UPGRADING],
    STOP: [APPLICATION_STATUSES.ACTIVE, APPLICATION_STATUSES.ERROR],
    START: [APPLICATION_STATUSES.STOPPED],
};

export const ApplicationFieldNames = {
    name: 'name',
    status: 'status',
    endpoints: 'endpoints',
};

export class Application {
    deployments: Deployment[] = [];
    description: string = '';
    id: string = '';
    application_id: string = '';
    app_id: string = '';
    name: string = '';
    status: string = '';
    status_detail: any;
    ctime: number = 0;
    mtime: number = 0;
    jira_issues: string[] = [];

    deployments_active: number = 0;
    deployments_error: number = 0;
    deployments_init: number = 0;
    deployments_stopped: number = 0;
    deployments_stopping: number = 0;
    deployments_terminated: number = 0;
    deployments_terminating: number = 0;
    deployments_waiting: number = 0;
    endpoints: string[] = [];

    // Ultimately we will start decorating data with annotations
    // For now this piece of code will just extend things.
    constructor(data?) {
        if (typeof data === 'object') {
            for (let key in data) {
                if (data.hasOwnProperty(key)) {
                    // Cast to deployment explicitly
                    if (key === 'deployments' && data[key] && data[key].length) {
                        let arr = [];
                        data[key].forEach((item) => {
                            if (!(item instanceof Deployment)) {
                                item = new Deployment(item);
                            }
                            arr.push(item);
                        });
                        this[key] = arr;
                    } else {
                        this[key] = data[key];
                    }

                }
            }
        }
    }

    public canStart(): boolean {
        return this.status !== APPLICATION_STATUSES.TERMINATED && this.status === APPLICATION_STATUSES.STOPPED;
    }

    public canStop(): boolean {
        return this.status !== APPLICATION_STATUSES.TERMINATED && this.status !== APPLICATION_STATUSES.STOPPED;
    }

    public canTerminate(): boolean {
        return this.status !== APPLICATION_STATUSES.TERMINATED;
    }

    public getAllExternalRoutes(): ExternalRoute[] {
        let arr: ExternalRoute[] = [];
        this.deployments.forEach((deployment) => {
            let a: ExternalRoute[] = deployment.template.external_routes || [];
            a.forEach((item: ExternalRoute) => {
                arr.push(item);
            });
        });
        return arr;
    }

    /**
     * Returns list of application issues combined with app deploment issues.
     */
    public get allJiraIssues() {
        let issues = this.jira_issues || [];
        (this.deployments || []).forEach(item => {
            issues = issues.concat(item.jira_issues || []);
        });
        return issues;
    }

    public totalDeployments(): number {
        return this.deployments_active + this.deployments_error
            + this.deployments_init + this.deployments_stopped + this.deployments_stopping
            + this.deployments_terminated + this.deployments_terminating
            + this.deployments_waiting;
    }

    get activePercentValue(): number {
        if (this.totalDeployments() === 0) {
            return 0;
        }

        return Math.round((this.deployments_active / this.totalDeployments()) * 100);
    }

    get errorPercentValue(): number {
        if (this.deployments.length === 0) {
            return 0;
        }

        return Math.round((this.deployments_error / this.totalDeployments()) * 100);
    }

    get stoppedPercentValue(): number {
        if (this.totalDeployments() === 0) {
            return 0;
        }

        return Math.round((this.deployments_stopped / this.totalDeployments()) * 100);
    }

    get daysActive(): number {
        let today = new Date();
        let date = new Date(this.ctime * 1000);

        return Math.round(Math.abs((today.getTime() - date.getTime()) / (24 * 60 * 60 * 1000)));
    }

    get daysLastDeployed(): number {
        let today = new Date();
        let date = new Date(this.mtime * 1000);

        return Math.round(Math.abs((today.getTime() - date.getTime()) / (24 * 60 * 60 * 1000)));
    }
}
