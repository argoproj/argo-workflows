import { Injectable } from '@angular/core';
import { Project } from '../model';
import { Http, URLSearchParams, Headers } from '@angular/http';

@Injectable()
export class ProjectService {

    constructor(private http: Http) {}

    public getProjects(params: {
            published?: boolean,
            repo?: string,
            branch?: string,
            search?: string,
            repo_branch?: string | { repo: string, branch: string }[],
        } = { }): Promise<Project[]> {
        let search = new URLSearchParams();
        if (params.published !== undefined && params.published !== null) {
            search.set('published', params.published ? 'true' : 'false');
        }
        if (params.repo) {
            search.set('repo', params.repo);
        }
        if (params.branch) {
            search.set('branch', params.branch);
        }
        if (params.search) {
            search.set('search', params.search);
        }
        if (params.repo_branch) {
            if (typeof params.repo_branch === 'string') {
                search.set('repo_branch', params.repo_branch);
            } else {
                search.set('repo_branch', JSON.stringify(params.repo_branch));
            }
        }
        return this.http.get('v1/projects', { search }).toPromise()
            .then(res => <Project[]>res.json().data)
            .then(projects => projects.map(this.convertProject.bind(this)));
    }

    public getProjectsByCategory( params: {
                published?: boolean,
                repo?: string,
                branch?: string,
                repo_branch?: string | { repo: string, branch: string }[],
            } = { }): Promise<Map<string, Project[]>> {
        return this.getProjects(params).then(projects => {
            let categoryToProject = new Map<string, Project[]>();
            projects.forEach(project => project.categories.forEach(category => {
                let categoryProjects = categoryToProject.get(category) || [];
                categoryToProject.set(category, categoryProjects);
                categoryProjects.push(project);
            }));
            return categoryToProject;
        });
    }

    public getProjectById(id: string, hideLoader = false): Promise<Project> {
        return this.http.get(`v1/projects/${id}`, { headers: new Headers({ isUpdated: hideLoader }) })
            .toPromise()
            .then(res => <Project> res.json())
            .then(this.convertProject.bind(this));
    }

    private convertProject(project: Project): Project {
        if (project.labels && project.labels.tags) {
            project.labels.tags = JSON.parse(project.labels.tags);
        }
        if (project.assets && project.assets.icon) {
            project.assets.icon = `/v1/projects/${project.id}/icon`;
        }
        return project;
    }
}
