import { Template } from './template';
import { Pod } from './pod';
import { ExternalRoute, TerminationPolicy } from '.';

export const DEPLOYMENT_STATUSES = {
    'INIT': 'Init',
    'WAITING': 'Waiting',
    'ACTIVE': 'Active',
    'ERROR': 'Error',
    'TERMINATING': 'Terminating',
    'TERMINATED': 'Terminated',
    'STOPPED': 'Stopped',
    'STOPPING': 'Stopping'
};

export const DeploymentFieldNames = {
    name: 'name',
    status: 'status',
};

export class Deployment {
    annotations: string[];
    app_generation: string;
    app_id: string;
    app_name: string;
    costid: any[];
    cpu: number;
    create_time: number;
    deployment_id: string;
    description: string;
    end_time: number;
    id: string;
    instances: Pod[] = [];
    labels: any = {};
    launch_time: number;
    mem: number;
    name: string;
    parameters: any[];
    run_time: number;
    status: string;
    status_detail: any;
    task_id: string;
    template: Template;
    template_id: string;
    user: string;
    wait_time: number;
    termination_policy: TerminationPolicy;
    jira_issues: string[];
    previous_deployment_id?: string;

    constructor(data?) {
        if (typeof data === 'object') {
            for (let key in data) {
                if (data.hasOwnProperty(key)) {
                    this[key] = data[key];
                }
            }
        }
    }

    public canStop(): boolean {
        let arr = [DEPLOYMENT_STATUSES.TERMINATED,
        DEPLOYMENT_STATUSES.TERMINATING,
        DEPLOYMENT_STATUSES.STOPPED,
        DEPLOYMENT_STATUSES.STOPPING];
        return arr.indexOf(this.status) > -1 ? false : true;
    }

    public canTerminate(): boolean {
        let arr = [DEPLOYMENT_STATUSES.TERMINATED];
        return arr.indexOf(this.status) > -1 ? false : true;
    }

    public canStart(): boolean {
        return this.status === DEPLOYMENT_STATUSES.STOPPED;
    }

    public canChangeScale(): boolean {
        return this.status !== DEPLOYMENT_STATUSES.STOPPED && this.status !== DEPLOYMENT_STATUSES.TERMINATED;
    }

    public getDeplymentLabels() {
        let l = [];
        for (let key in this.labels) {
            if (this.labels[key]) {
                l.push({ key, value: this.labels[key] });
            }
        }
        return l;
    }

    public getExternalRoutes(): ExternalRoute[] {
        let arr: ExternalRoute[] = [];
        if (this.template && this.template.external_routes && this.template.external_routes.length > 0) {
            arr = this.template.external_routes;
        }
        return arr;
    }

    public getFirstExternalRoute(): string {
        let arr: ExternalRoute[] = this.getExternalRoutes();
        return arr.length > 0 ? arr[0].dns_name : '';
    }
}
