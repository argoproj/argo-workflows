import { Injectable, EventEmitter } from '@angular/core';
import { Http, URLSearchParams, Headers } from '@angular/http';
import { Observable, Observer, Subscription } from 'rxjs';

import { AxHeaders } from './headers';

import { ITool, ContainerRegistry, SamlConfigTool, CertificateTool, Route53Config, ToolCategory} from '../model';

const PRECONFIGURED_URLS = [
    'https://github.com/Applatix/appstore.git',
    'https://github.com/Applatix/example-approval.git',
    'https://github.com/Applatix/example-dynamic-fixtures.git'
];

@Injectable()
export class ToolService {

    public onToolsChanged: EventEmitter<any> = new EventEmitter();

    private recentJiraConfigCheck: Promise<ITool>;
    private jiraConfigChanges: Observable<ITool>;

    constructor(private _http: Http) {
        this.jiraConfigChanges = Observable.create((observer: Observer<ITool>) => {
            let subscription = this.onToolsChanged.subscribe(async () => {
                let config = await this.loadJiraConfig();
                this.recentJiraConfigCheck = Promise.resolve(config);
                observer.next(config);
            }) as Subscription;
            return () => subscription.unsubscribe();
        }).share();
    }

    getJiraConfig(): Observable<ITool> {
        if (!this.recentJiraConfigCheck) {
            this.recentJiraConfigCheck = this.loadJiraConfig();
        }
        return Observable.merge(Observable.fromPromise(this.recentJiraConfigCheck), this.jiraConfigChanges);
    }

    isJiraConfigured(): Observable<boolean> {
        return this.getJiraConfig().map(config => !!config);
    }

    loadJiraConfig(): Promise<ITool> {
        return this.getToolsAsync({type: 'jira'}, true).toPromise().then(result => result.data.length > 0 ? result.data[0] : null);
    }

    getToolsAsync(params: { category?: ToolCategory, type?: string } = {}, hideLoader?: boolean): Observable<{ data: any[] }> {
        let filter = new URLSearchParams();
        let headers = new Headers();

        if (params.category) {
            filter.set('category', params.category.toString());
        }

        if (params.type) {
            filter.set('type', params.type.toString());
        }

        if (hideLoader) {
            headers.append('isUpdated', hideLoader.toString());
        }

        return this._http.get('v1/tools', { headers: headers, search: filter }) .map(res => res.json());
    }

    /**
     *
     * Get the system's SAML information for admin to use
     * The admin will use this data to configure the Argo app
     * in the org SSO solution like OKTA
     */
    getSAMLInfo() {
        return this._http.get(`v1/auth/saml/info`)
            .map(res => res.json());
    }

    getToolByIdAsync(toolId) {
        return this._http.get(`v1/tools/${toolId}`)
            .map(res => res.json());
    }

    updateToolAsync(tool: ITool) {
        return this._http.put(`v1/tools/${tool.id}`, JSON.stringify(tool))
            .map(res => {
                this.onToolsChanged.emit();
                return res.json();
            });
    }

    createToolAsync(tool: ITool, isPublic: boolean) {
        return this._http.post(`v1/tools&public=${isPublic}`, JSON.stringify(tool))
            .map(res => {
                this.onToolsChanged.emit();
                return res.json();
            });
    }

    createSamlConfigTool(samlConfig: SamlConfigTool) {
        return this._http.post(`v1/tools/authentication/saml`, JSON.stringify(samlConfig))
            .map(res => res.json());
    }

    createCertificateTool(tool: CertificateTool) {
        return this._http.post(`v1/tools/certificate/server`, JSON.stringify(tool))
            .map(res => res.json());
    }

    deleteToolAsync(toolId: string) {
        return this._http.delete(`v1/tools/${toolId}`)
            .map(res => {
                this.onToolsChanged.emit();
                return res.json();
            });
    }

    connectAccountAsync(tool: ITool) {
        return this._http.post(`v1/tools`, JSON.stringify(tool), { headers: new AxHeaders({ noErrorHandling: true }) })
            .map(res => {
                this.onToolsChanged.emit();
                return res.json();
            });
    }

    testCredentialsAsync(tool: ITool) {
        return this._http.post(`v1/tools/test`, JSON.stringify(tool), { headers: new AxHeaders({ noErrorHandling: true }) })
            .map(res => {
                this.onToolsChanged.emit();
                return res.json();
            });
    }

    postContainerRegistry(containerRegistry: ContainerRegistry) {
        return this._http.post(`v1/tools/registry/${containerRegistry.type}`, JSON.stringify(containerRegistry), { headers: new AxHeaders({ noErrorHandling: true }) })
            .map(res => res.json());
    }

    postToDomainManagement(domainObject?: Route53Config) {
        if (domainObject) {
            return this._http.put(`v1/tools/${domainObject.id}`, domainObject).map(res => new Route53Config(res.json()));
        } else {
            return this._http.post('v1/tools/domain_management/route53', {}).map(res => new Route53Config(res.json()));
        }
    }

    isScmConfigured(): Promise<boolean> {
        return this.getToolsAsync({ category: 'scm' }).map(res => res.data).toPromise().then(tools => {
            return tools.filter(tool => PRECONFIGURED_URLS.indexOf(tool.url) === -1).length > 0;
        });
    }
}
