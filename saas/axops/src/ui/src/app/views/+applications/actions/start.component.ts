import { Component, Input, Output, EventEmitter } from '@angular/core';

import { Application, Deployment } from '../../../model';
import { ApplicationsService, NotificationService, DeploymentsService, ModalService } from '../../../services';

@Component({
    selector: 'ax-start-action',
    template: `<a (click)="doStartAction($event)" class="action-icon action-with-icon" [ngClass]="{'action-with-icon--start': showIcon}">
                <i *ngIf="showIcon"class="ax-icon-start" aria-hidden="true"></i>
                START
                </a>`,
    styles: [ require('./actions.scss') ],
})
export class StartComponent {
    @Input()
    application: Application;
    @Input()
    deployment: Deployment;
    @Input()
    showIcon: boolean = true;

    @Output()
    onStart: EventEmitter<Application | Deployment> = new EventEmitter<Application | Deployment>();

    constructor(
        private applicationsService: ApplicationsService,
        private notificationService: NotificationService,
        private deploymentService: DeploymentsService,
        private modalService: ModalService
    ) {
    }
    public doStartAction($event) {

        if (this.application && this.deployment) {
            throw 'Config error: The component supports either Application or Deployment for a given instance. Not both';
        }

        let type: string = this.application ? 'Application' : 'Deployment';
        let itemName: string = this.application ? this.application.name : this.deployment.name;

        this.modalService.showModal(`Confirm ${type} Start `, `Are you sure you want to start the ${type}: “${itemName}“?`)
            .subscribe(result => {
                if (result) {
                    if (this.application) {
                        this.startApplication();
                    }

                    if (this.deployment) {
                        this.startDeployment();
                    }
                }
            });

        $event.stopPropagation();
    }

    private startApplication() {
        this.applicationsService.startApplication(this.application.id, true).subscribe(() => {
            // do nothing
        }, (err) => {
            this.notificationService.showNotification.emit(
                { message: err.message });
        });

        setTimeout(() => {
            this.onStart.emit(this.application);
        }, 1000);
    }

    private startDeployment() {
        this.deploymentService.startDeployment(this.deployment.id, true).subscribe(() => {
            // do nothing
        }, (err) => {
            this.notificationService.showNotification.emit(
                { message: err.message });
        });

        setTimeout(() => {
            this.onStart.emit(this.deployment);
        }, 1000);
    }
}
