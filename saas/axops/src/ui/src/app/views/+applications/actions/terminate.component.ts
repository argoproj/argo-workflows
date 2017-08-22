import { Component, Input, Output, EventEmitter } from '@angular/core';

import { Application, Deployment } from '../../../model';
import { ApplicationsService, NotificationService, DeploymentsService, ModalService } from '../../../services';

@Component({
    selector: 'ax-terminate-action',
    template: `<a (click)="doTerminateAction($event)" class="action-icon action-with-icon" [ngClass]="{'action-with-icon--terminate': showIcon}">
                <i class="ax-icon-terminate" aria-hidden="true" *ngIf="showIcon"></i>
                TERMINATE
                </a>`,
    styles: [ require('./actions.scss') ],
})

export class TerminateComponent {
    @Input()
    application: Application;
    @Input()
    deployment: Deployment;

    @Input()
    showIcon: boolean = true;

    @Output()
    onTerminate: EventEmitter<Application | Deployment> = new EventEmitter<Application | Deployment>();

    constructor(
        private applicationsService: ApplicationsService,
        private notificationService: NotificationService,
        private deploymentService: DeploymentsService,
        private modalService: ModalService
    ) {
    }
    public doTerminateAction($event) {
        if (this.application && this.deployment) {
            throw 'Config error: The component supports either Application or Deployment for a given instance. Not both';
        }

        let type: string = this.application ? 'Application' : 'Deployment';
        let itemName: string = this.application ? this.application.name : this.deployment.name;

        this.modalService.showModal(`Confirm ${type} Termination `, `Are you sure you want to terminate the ${type}: “${itemName}“?`)
            .subscribe(result => {
                if (result) {
                    if (this.application) {
                        this.terminateApplication();
                    }

                    if (this.deployment) {
                        this.terminateDeployment();
                    }
                }
            });
        $event.stopPropagation();
    }

    private terminateApplication() {
        this.applicationsService.deleteAppById(this.application.id, true).subscribe(() => {
            // do nothing
        }, (err) => {
            this.notificationService.showNotification.emit(
                { message: err.message });
        });

        setTimeout(() => {
            this.onTerminate.emit(this.application);
        }, 1000);
    }

    private terminateDeployment() {
        this.deploymentService.deleteDeploymentById(this.deployment.id, true).subscribe(() => {
            // do nothing
        }, (err) => {
            this.notificationService.showNotification.emit(
                { message: err.message });
        });
        setTimeout(() => {
            this.onTerminate.emit(this.deployment);
        }, 1000);

    }
}
