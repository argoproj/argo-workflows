import { Component, OnInit, OnDestroy, ViewChild } from '@angular/core';
import { Router } from '@angular/router';
import { Subscription } from 'rxjs';

import { Project } from '../../../model';
import { LaunchPanelService } from '../../../common/multiple-service-launch-panel/launch-panel.service';
import { MultipleServiceLaunchPanelComponent } from '../../../common/multiple-service-launch-panel/multiple-service-launch-panel.component';
import { PlaygroundProjects, PlaygroundInfoService, SystemService, AuthorizationService, ProjectService, AuthenticationService } from '../../../services';

@Component({
    selector: 'ax-playground',
    templateUrl: './playground.html',
    styles: [ require('./playground.scss') ],
})
export class PlaygroundComponent implements OnInit, OnDestroy {

    @ViewChild(MultipleServiceLaunchPanelComponent)
    public multipleServiceLaunchPanel: MultipleServiceLaunchPanelComponent;

    public projects: PlaygroundProjects;
    public selectedProject: Project;
    public version: string;
    public showVideoPopup: boolean;

    get showLaunchProjectPanel() {
        return this.selectedProject && !this.launchPanelService.isPanelVisible();
    }

    private subscriptions: Subscription[] = [];

    constructor(
        private launchPanelService: LaunchPanelService,
        private playgroundInfoService: PlaygroundInfoService,
        private authorizationService: AuthorizationService,
        private authenticationService: AuthenticationService,
        private systemService: SystemService,
        private projectService: ProjectService,
        private router: Router) {
    }

    public ngOnInit() {
        this.playgroundInfoService.loadPlaygroundProjects().then(projects => this.projects = projects);
        this.launchPanelService.initPanel(this.multipleServiceLaunchPanel);
        this.subscriptions.push(this.launchPanelService.onLaunch.subscribe(tasks => {
            if (tasks.length > 0) {
                this.playgroundInfoService.startPlaygroundTask(this.selectedProject, tasks[0]);
            }
            this.selectedProject = null;
        }));

        this.systemService.getVersion().toPromise().then(info => this.version = info.version.split('-')[0]);
    }

    public completePlayground() {
        this.authorizationService.completeIntroduction();
    }

    public logout() {
        this.authenticationService.logout();
    }

    public viewCashboard() {
        this.playgroundInfoService.startPlaygroundTask(null, null);
        this.router.navigateByUrl('/app/cashboard');
    }

    public ngOnDestroy() {
        this.subscriptions.forEach(item => item.unsubscribe());
        this.subscriptions = [];
    }

    public selectProject(project: Project) {
        if (project) {
            // Reload project to get all project assets
            this.projectService.getProjectById(project.id).then(res => {
                this.selectedProject = res;
            });
        } else {
            this.selectedProject = null;
        }
    }
}
