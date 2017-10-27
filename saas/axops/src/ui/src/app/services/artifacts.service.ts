import { Observable } from 'rxjs';
import { Inject, Injectable } from '@angular/core';
import { Http, URLSearchParams, Headers } from '@angular/http';
import { Artifact } from '../model';

@Injectable()
export class ArtifactsService {
    constructor(@Inject(Http) private _http) {
    }

    public getArtifacts(params?: {
        workflow_id?: string,
        service_instance_id?: string,
        tags?: string,
        retention_tags?: string,
        artifact_type?: string,
        name?: string,
    }, hideLoader?: boolean): Observable<Artifact[]> {
        let filter = new URLSearchParams();
        let headers = new Headers();

        filter.set('action', 'search');
        if (params.workflow_id) {
            filter.set('workflow_id', params.workflow_id.toString());
        }
        if (params.service_instance_id) {
            filter.set('service_instance_id', params.service_instance_id.toString());
        }
        if (params.tags) {
            filter.set('tags', params.tags.toString());
        }
        if (params.retention_tags) {
            filter.set('retention_tags', params.retention_tags.toString());
        }
        if (params.artifact_type) {
            filter.set('artifact_type', params.artifact_type.toString());
        }

        if (params.name) {
            filter.set('name', params.name.toString());
        }

        if (hideLoader) {
            headers.append('isUpdated', hideLoader.toString());
        }

        return this._http.get('v1/artifacts', {headers: headers, search: filter})
            .map(res => res.json().data.map(item => new Artifact(item)));
    }

    public getArtifactById(artifactId) {
        return this._http.get(`v1/artifacts/${artifactId}`)
            .map(res => res.json());
    }

    public browseArtifact(params: {
        artifact_id: string
    }, hideLoader?: boolean) {
        let headers = new Headers();

        if (hideLoader) {
            headers.append('isUpdated', hideLoader.toString());
        }

        return this._http.get(`v1/artifacts/${params.artifact_id}/browse`, {headers: headers})
            .map(res => res.json());
    }

    public getUsage(hideLoader?: boolean) {
        let filter = new URLSearchParams();
        let headers = new Headers();

        filter.set('action', 'get_usage');

        if (hideLoader) {
            headers.append('isUpdated', hideLoader.toString());
        }

        return this._http.get('v1/artifacts', {headers: headers, search: filter})
            .map(res => res.json());
    }

    public getArtifactTags(params?: { search?: string }, hideLoader?: boolean) {
        let filter = new URLSearchParams();
        let headers = new Headers();

        if (params && params.search) {
            filter.set('search', params.search.toString());
        }

        if (hideLoader) {
            headers.append('isUpdated', hideLoader.toString());
        }

        return this._http.get(`v1/artifacts?action=list_tags`, {headers: headers, search: filter})
            .map(res => res.json());
    }

    public tagOperation(action: 'tag' | 'untag', params: {
        workflow_ids: string,
        tag: string,
    }, hideLoader?: boolean) {
        let filter = new URLSearchParams();
        let headers = new Headers();
        if (hideLoader) {
            headers.append('isUpdated', hideLoader.toString());
        }

        filter.set('action', action);
        if (params.workflow_ids) {
            filter.set('workflow_ids', params.workflow_ids.toString());
        }
        if (params.tag) {
            filter.set('tag', params.tag.toString());
        }

        return this._http.put(`v1/artifacts`, null, {headers: headers, search: filter.toString()})
            .map(res => res.json());
    }

    public cleanArtifacts(hideLoader?: boolean) {
        let filter = new URLSearchParams();
        let headers = new Headers();
        filter.set('action', 'clean');
        if (hideLoader) {
            headers.append('isUpdated', hideLoader.toString());
        }
        return this._http.put(`v1/artifacts`, null, {headers: headers, search: filter.toString()})
            .map(res => res.json());
    }

    public getArtifactDownloadUrl(artifactId: string): string {
        let filter = new URLSearchParams();
        filter.set('action', 'download');
        filter.append('artifact_id', artifactId);
        return `v1/artifacts?${filter.toString()}`;
    }

    public async getArtifactDownloadUrlByName(serviceInstanceId: string, name: string): Promise<string> {
        let artifacts = await this.getArtifacts({ service_instance_id: serviceInstanceId, name }).toPromise();
        if (artifacts.length > 0) {
            return this.getArtifactDownloadUrl(artifacts[0].artifact_id);
        }
        return null;
    }
}
