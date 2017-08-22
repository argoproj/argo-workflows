import { EventEmitter } from '@angular/core';
import { ModalService } from '../../services';
import { NotificationsService } from 'argo-ui-lib/src/components';

interface Action<T> {
    title: string;
    execute(item: T): Promise<any>;
    isApplicable(item: T): boolean;
    warningMessage(notApplicableItems: T[]): string;
    confirmation(count: number): string;
    postMessage(successfulCount: number, failedCount: number): string;
}

export class BulkUpdater<T> {

    private actions = new Map<string, Action<T>>();
    private allItems: T[] = [];
    private selectedItems: T[] = [];

    public readonly actionExecuted = new EventEmitter<string>();

    constructor(private modalService: ModalService, private notificationsService: NotificationsService) {
    }

    public get items(): T[] {
        return this.allItems;
    }

    public set items(val: T[]) {
        this.allItems = val;
        this.selectedItems = [];
    }

    public toggleItemSelection(item: T) {
        let index = this.selectedItems.indexOf(item);
        if (index > -1) {
            this.selectedItems.splice(index, 1);
        } else {
            this.selectedItems.push(item);
        }
    }

    public addAction(name: string, action: Action<T>): BulkUpdater<T> {
        this.actions.set(name, action);
        return this;
    }

    public noApplicableSelected(actionName: string) {
        let action = this.getAction(actionName);
        return !this.selectedItems.find(item => action.isApplicable(item));
    }

    public toggleAllSelection() {
        if (this.isAllSelected) {
            this.selectedItems = [];
        } else {
            this.selectedItems = this.items.slice();
        }
    }

    public isSelected(item: T) {
        return this.selectedItems.indexOf(item) > -1;
    }

    public get isAllSelected(): boolean {
        return this.items.length > 0 && this.selectedItems.length === this.items.length;
    }

    public get selectedCount(): number {
        return this.selectedItems.length;
    }

    public clearSelection() {
        this.selectedItems = [];
    }

    public execute(actionName: string) {
        let action = this.getAction(actionName);
        let notApplicableItems = this.selectedItems.filter(item => !action.isApplicable(item));
        let warning = '';
        if (notApplicableItems.length) {
            warning = action.warningMessage(notApplicableItems);
        }
        this.modalService.showModal(action.title, action.confirmation(this.selectedItems.length), warning, { name: warning ? 'fa fa-exclamation-triangle' : '', color: 'warning' } )
            .subscribe(confirmed => {
                if (confirmed) {
                    Promise.all(this.selectedItems.filter(item => action.isApplicable(item)).map(async item => {
                        try {
                            await action.execute(item);
                            return true;
                        } catch (e) {
                            return false;
                        }
                    })).then(res => {
                        let successful = res.filter(item => item).length;
                        let failed = res.length - successful;
                        let postMessage = action.postMessage(successful, failed);
                        if (failed > 0) {
                            this.notificationsService.warning(postMessage);
                        } else {
                            this.notificationsService.success(postMessage);
                        }
                        this.actionExecuted.emit(actionName);
                        this.clearSelection();
                    });
                }
            });
    }

    private getAction(actionName: string) {
        let action = this.actions.get(actionName);
        if (!action) {
            throw new Error(`Action '${actionName}' is not supported`);
        }
        return action;
    }
}
