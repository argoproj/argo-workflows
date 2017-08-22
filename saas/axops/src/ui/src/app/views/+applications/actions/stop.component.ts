import { Component, Input, Output, EventEmitter } from '@angular/core';

import { Application, Deployment } from '../../../model';
import { ApplicationsService, NotificationService, DeploymentsService, ModalService } from '../../../services';

@Component({
    selector: 'ax-stop-action',
    template: `<a (click)="doStopAction($event)" class="action-icon action-with-icon" [ngClass]="{'action-with-icon--stop': showIcon}">
                <i *ngIf="showIcon" class="ax-icon-stop" aria-hidden="true"></i>
                STOP
                </a>`,
    styles: [ require('./actions.scss') ],
})
export class StopComponent {
    @Input()
    application: Application;
    @Input()
    deployment: Deployment;
    @Input()
    showIcon: boolean = true;

    @Output()
    onStop: EventEmitter<Application | Deployment> = new EventEmitter<Application | Deployment>();

    constructor(
        private applicationsService: ApplicationsService,
        private notificationService: NotificationService,
        private deploymentService: DeploymentsService,
        private modalService: ModalService
    ) {
    }

    public doStopAction($event) {
        if (this.application && this.deployment) {
            throw 'Config error: The component supports either Application or Deployment for a given instance. Not both';
        }

        let type: string = this.application ? 'Application' : 'Deployment';
        let itemName: string = this.application ? this.application.name : this.deployment.name;

        this.modalService.showModal(`Confirm ${type} Stop `, `Are you sure you want to stop the ${type}: “${itemName}“?`)
            .subscribe(result => {
                if (result) {
                    if (this.application) {
                        this.stopApplication();
                    }

                    if (this.deployment) {
                        this.stopDeployment();
                    }
                }
            });
        $event.stopPropagation();
    }

    private stopApplication() {
        this.applicationsService.stopApplication(this.application.id, true).subscribe(() => {
            // do nothing
        }, (err) => {
            this.notificationService.showNotification.emit(
                { message: err.message });
        });

        setTimeout(() => {
            this.onStop.emit(this.application);
        }, 1000);
    }

    private stopDeployment() {
        this.deploymentService.stopDeployment(this.deployment.id, true).subscribe(() => {
            // do nothing
        }, (err) => {
            this.notificationService.showNotification.emit(
                { message: err.message });
        });
        setTimeout(() => {
            this.onStop.emit(this.deployment);
        }, 1000);

    }

}
