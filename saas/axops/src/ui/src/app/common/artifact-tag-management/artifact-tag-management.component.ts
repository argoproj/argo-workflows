import { Component, Input, Output, OnInit, OnDestroy, EventEmitter } from '@angular/core';
import { FormGroup, FormControl, Validators } from '@angular/forms';
import { Subscription } from 'rxjs/Subscription';
import { Observable } from 'rxjs/Observable';

import { NotificationsService } from 'argo-ui-lib/src/components';
import { ArtifactsService } from '../../services';
import { CustomRegex } from '../customValidators/CustomRegex';

@Component({
    selector: 'ax-artifact-tag-management',
    templateUrl: './artifact-tag-management.component.html',
    styles: [ require('./artifact-tag-management.scss') ],
})
export class ArtifactTagManagementComponent implements OnInit, OnDestroy {
    @Input()
    public set show(value) {
        if (value) {
            this.getArtifactsTags();
            this.showSidePanel = value;
        }
    }

    @Input()
    public workflowId: string;

    @Input()
    public usedTags: string[];

    @Output()
    public onClose: EventEmitter<any> = new EventEmitter();

    @Output()
    public onApply: EventEmitter<any> = new EventEmitter();

    public submitted: boolean;

    public artifactTags: string[];
    public showSidePanel: boolean = false;
    public isVisibleManageArtifactTags: boolean = false;
    public dataLoaded: boolean = false;
    public artifactTagNamePattern: string;
    private newListOfUsedTags: string[] = [];
    private subscriptions: Subscription[] = [];
    private addArtifactTagForm: FormGroup;

    constructor(private artifactsService: ArtifactsService,
        private notificationsService: NotificationsService) {
        this.artifactTagNamePattern = CustomRegex.artifactTagName;
    }

    public ngOnInit() {
        this.addArtifactTagForm = new FormGroup({
            artifact_tag: new FormControl('', Validators.pattern(CustomRegex.artifactTagName)),
        });
    }

    public ngOnDestroy() {
        this.subscriptions.forEach(subscribe => subscribe.unsubscribe());
    }

    public getArtifactsTags() {
        this.subscriptions.push(this.artifactsService.getArtifactTags().subscribe(
            (success: any) => {
                this.dataLoaded = true;
                this.artifactTags = success.data;
                // deep copy of table
                this.newListOfUsedTags = this.usedTags.slice();
            }, error => {
                this.dataLoaded = true;
            }
        ));
    }

    public tagIsUsed(tag): boolean {
        return this.newListOfUsedTags.indexOf(tag) !== -1;
    }

    public toggleTag(tag: string) {
        if (this.newListOfUsedTags.indexOf(tag) === -1) {
            this.newListOfUsedTags.push(tag);
            this.addArtifactTagForm.controls['artifact_tag'].setValue('');
        } else {
            this.newListOfUsedTags.splice(this.newListOfUsedTags.indexOf(tag), 1);
        }
    }

    public apply() {
        let observableList: Observable<any>[] = [];
        let unTagList: string[] = [];
        let addTagList: string[] = [];
        this.artifactTags.forEach((tag: string) => {
            if (this.newListOfUsedTags.indexOf(tag) === -1 && this.usedTags.indexOf(tag) !== -1) {
                unTagList.push(tag);
            } else if (this.newListOfUsedTags.indexOf(tag) !== -1 && this.usedTags.indexOf(tag) === -1) {
                addTagList.push(tag);
            }
        });
        if (unTagList.length > 0) {
            observableList.push(this.artifactsService.tagOperation('untag',
                { workflow_ids: this.workflowId, tag: unTagList.join(',') },
                true));
        }
        if (addTagList.length > 0) {
            observableList.push(this.artifactsService.tagOperation('tag',
                { workflow_ids: this.workflowId, tag: addTagList.join(',') },
                true));
        }

        this.subscriptions.push(Observable.forkJoin(observableList).subscribe(success => {
            this.onApply.emit();
            this.notificationsService.success(`Artifact tags updated.`);
        }, error => {
            this.notificationsService.internalError();
        }
        ));

        this.close();
    }

    public close() {
        this.showSidePanel = false;
        this.newListOfUsedTags = [];
        this.dataLoaded = false;
        this.onClose.emit();
        this.addArtifactTagForm.reset();
    }

    public openManageArtifactsTags() {
        this.isVisibleManageArtifactTags = true;
    }

    public cancelManage() {
        this.isVisibleManageArtifactTags = false;
    }

    public addNewArtifactTag() {
        this.submitted = true;
        if (this.addArtifactTagForm.valid) {
            let inputString = this.addArtifactTagForm.value['artifact_tag'];
            if (inputString && this.artifactTags.indexOf(inputString) === -1) {
                this.artifactTags.push(inputString);
                this.newListOfUsedTags.push(inputString);
                this.addArtifactTagForm.controls['artifact_tag'].setValue('');
            } else if (inputString && this.newListOfUsedTags.indexOf(inputString) === -1) {
                this.newListOfUsedTags.push(inputString);
                this.addArtifactTagForm.controls['artifact_tag'].setValue('');
            }
        }
    }
}
