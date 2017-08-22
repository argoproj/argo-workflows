import { Component, Input, ViewChild } from '@angular/core';

import { Project, ProjectAction } from '../../../model';
import { CommitsService, ProjectService } from '../../../services';

import { LaunchPanelService } from '../../../common/multiple-service-launch-panel/launch-panel.service';
import { NotificationsService } from 'argo-ui-lib/src/components';

@Component({
    selector: 'ax-project-launch-button',
    templateUrl: './project-launch-button.html',
    styles: [ require('./project-launch-button.scss') ],
})
export class ProjectLaunchButtonComponent {

    @Input()
    public set project(val: Project) {
        this.projectInfo = val;
        this.actions = Object.keys(val && val.actions || {}).map(name => Object.assign({}, { name }, val.actions[name]));
    }

    public get project(): Project {
        return this.projectInfo;
    }

    public actions: (ProjectAction & { name: string })[] = [];

    @ViewChild('actionsDropDown')
    public actionsDropDown;

    public loading: boolean;

    private projectInfo: Project;

    constructor(
        private launchPanelService: LaunchPanelService,
        private commitsService: CommitsService,
        private notificationsService: NotificationsService,
        private projectService: ProjectService) {}

    public launchProject(action: ProjectAction) {
        if (!this.loading) {
            this.loading = true;
            this.commitsService.getCommitsAsync({
                repo: this.project.repo,
                branch: this.project.branch,
                limit: 1,
            }).toPromise().then(res => {
                if (res.data.length === 0) {
                    this.loading = false;
                    this.notificationsService.error('Cannot load data from project repository');
                } else {
                    this.projectService.getProjectById(this.project.id).then(project => {
                        this.launchPanelService.openPanel(res.data[0], { project, action }, false);
                        this.loading = false;
                    });
                }
            });
        }
    }
}
