import { Injectable, EventEmitter } from '@angular/core';

import { FixtureInstance, FixtureClass, FixtureStatuses, FixtureActions } from '../../model';
import { CapitalizePipe } from '../../pipes';
import { FixtureService } from '../../services';
import { DropdownMenuSettings, NotificationsService } from 'argo-ui-lib/src/components';

@Injectable()
export class FixturesViewService {

    public fixtureUpdated = new EventEmitter<FixtureInstance>();

    constructor(private fixtureService: FixtureService, private notificationsService: NotificationsService) {
    }

    public getInstanceActionMenu(fixtureInstance: FixtureInstance, fixtureClass: FixtureClass, launchers: {
        customActionLauncher: (name: string, allParamsHasValues: boolean) => any,
        cloneLaucher: () => any,
        editLaucher: () => any
    }): DropdownMenuSettings {
        let classActionDefinitions = fixtureClass.actions || {};

        let hasDeleteAction = !!classActionDefinitions['delete'];

        let editAction = { title: 'Edit', action: () => launchers.editLaucher(), iconName: 'fa-pencil' };
        let cloneAction = { title: 'Duplicate', action: () => launchers.cloneLaucher(), iconName: 'fa-clone' };
        let recreateAction = { title: 'Retry Create', action: () => this.retryCreate(fixtureInstance), iconName: 'ax-icon-fixturenew' };
        let deleteAction = { title: 'Delete', action: () => this.startDeleteAction(fixtureInstance), iconName: 'fa-trash-o' };
        let markDeletedAction = { title: 'Mark Deleted', action: () => this.markDeleted(fixtureInstance), iconName: 'fa-trash-o' };
        let enableAction = { title: 'Enable', action: () => this.setEnabled(fixtureInstance, true), iconName: 'fa-toggle-on' };
        let disableAction = { title: 'Disable', action: () => this.setEnabled(fixtureInstance, false), iconName: 'fa-toggle-off' };
        let maintanenceActions = Object.keys(classActionDefinitions).filter(action => action !== FixtureActions.CREATE && action !== FixtureActions.DELETE).map(actionName => {
            let parameters = classActionDefinitions[actionName].parameters || {};
            let allParamsHasValues = Object.keys(parameters).map(paramName => !!parameters[paramName]).reduce((first, second) => first && second, true);
            return {
                title: new CapitalizePipe().transform(actionName, []),
                action: () => launchers.customActionLauncher(actionName, allParamsHasValues),
                iconName: 'fa-cog',
            };
        });

        let items: { title: string, iconName: string, action: () => any }[] = [editAction, cloneAction];

        switch (fixtureInstance.status) {
            case FixtureStatuses.CREATE_ERROR:
                items = items.concat([recreateAction, deleteAction, markDeletedAction]);
                break;
            case FixtureStatuses.ACTIVE:
                if (hasDeleteAction) {
                    items = items.concat([deleteAction]);
                }
                items = items.concat([markDeletedAction]).concat(maintanenceActions);
                break;
            case FixtureStatuses.DELETE_ERROR:
                items = items.concat([Object.assign({}, deleteAction, { title: 'Retry Delete' }), markDeletedAction]).concat(maintanenceActions);
                break;
        }
        if (fixtureInstance.status !== FixtureStatuses.DELETED && fixtureInstance.status !== FixtureStatuses.OPERATING) {
            if (fixtureInstance.enabled) {
                items.unshift(disableAction);
            } else {
                items.unshift(enableAction);
            }
        }
        return new DropdownMenuSettings(items, 'fa-ellipsis-v');
    }

    private async retryCreate(fixtureInstance: FixtureInstance) {
        await this.fixtureService.runFixtureInstanceAction(fixtureInstance.id, FixtureActions.CREATE);
        this.notificationsService.success('Create job has been successfully started.');
        this.fixtureUpdated.emit(fixtureInstance);
    }

    private async startDeleteAction(fixtureInstance: FixtureInstance) {
        await this.fixtureService.deleteFixtureInstance(fixtureInstance.id);
        this.notificationsService.success('Delete job has been successfully started.');
        this.fixtureUpdated.emit(fixtureInstance);
    }

    private async markDeleted(fixtureInstance: FixtureInstance) {
        await this.fixtureService.setFixtureStatus(fixtureInstance.id, FixtureStatuses.DELETED);
        this.fixtureUpdated.emit(fixtureInstance);
    }

    private async setEnabled(fixtureInstance: FixtureInstance, isEnabled: boolean) {
        await this.fixtureService.setFixtureInstanceEnabled(fixtureInstance.id, isEnabled);
        this.fixtureUpdated.emit(fixtureInstance);
    }
}
