import { Injectable, EventEmitter } from '@angular/core';
import { MultipleServiceLaunchPanelComponent } from './multiple-service-launch-panel.component';
import { Artifact, Commit, Project, ProjectAction, Task, Template } from '../../model';
import { Subscription } from 'rxjs';

@Injectable()
export class LaunchPanelService {
    private static instance: LaunchPanelService;

    public onOpenPanel = new EventEmitter<any>();
    public onLaunch = new EventEmitter<any>();
    private panel: MultipleServiceLaunchPanelComponent;
    private launchSubscription: Subscription;

    public static create(): LaunchPanelService {
        if (!LaunchPanelService.instance) {
            LaunchPanelService.instance = new LaunchPanelService();
        }
        return LaunchPanelService.instance;
    }

    public initPanel(panel: MultipleServiceLaunchPanelComponent) {
        if (this.launchSubscription) {
            this.launchSubscription.unsubscribe();
        }
        this.panel = panel;
        this.launchSubscription = panel.submitted.subscribe(tasks => this.onLaunch.emit(tasks));
    }

    public openPanel(commit: Commit,
                    options?: Task | Template | { project: Project, action: ProjectAction },
                    withCommitOnly = true,
                    artifacts?: Artifact[],
                    resubmit: boolean = false) {
        this.panel.openPanel(commit, options, withCommitOnly, artifacts, resubmit);
        this.onOpenPanel.emit({});
    }

    public isPanelVisible() {
        return this.panel && this.panel.isVisibleSelectServiceTemplatesPanel;
    }
}
