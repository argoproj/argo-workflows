import { Component, Input, Output, EventEmitter, OnChanges, OnDestroy, SimpleChanges, ViewChildren, QueryList, ElementRef } from '@angular/core';
import { URLSearchParams } from '@angular/http';
import { Subscription } from 'rxjs';

import { Task, Deployment, Artifact, ARTIFACT_TYPES, DeletedStatus, TaskStatus } from '../../model';
import { JobTreeNode } from '../../common/workflow-tree/workflow-tree.view-models';
import { ArtifactsService } from '../../services';
import { SortOperations } from '../../common/sortOperations/sortOperations';
import { LaunchPanelService } from '../../common/multiple-service-launch-panel/launch-panel.service';

class SelectableArtifact extends Artifact {
    selected: boolean = false;
}

class ArtifactGroup {
    name: string = '';
    artifacts: SelectableArtifact[] = [];
}

@Component({
    selector: 'ax-artifacts',
    templateUrl: './artifacts.html',
    styles: [ require('./artifacts.scss') ],
})
export class ArtifactsComponent implements OnChanges, OnDestroy {
    protected readonly artifactTypeFilter: string[] = [
        ARTIFACT_TYPES.USER_LOG, ARTIFACT_TYPES.INTERNAL, ARTIFACT_TYPES.EXPORTED, ARTIFACT_TYPES.AX_LOG, ARTIFACT_TYPES.AX_LOG_EXTERNAL
    ];

    @Input()
    public allowSelecting: boolean = false;

    @Input()
    public task: Task;

    @Input()
    public set deployment(val: Deployment) {
        if (val && val.id) {
            this.loadArtifacts(null, val.id);
            this.flatMapOfSteps = [];
        }
    }

    @Output()
    public selectedCountChanged: EventEmitter<number> = new EventEmitter<number>();

    public artifactGroups: ArtifactGroup[] = [];
    @ViewChildren('a')
    public aElements: QueryList<ElementRef>;

    public dataLoaded: boolean = false;
    public isArtifactGroupsEmpty: boolean = true;
    public artifactTypes = ARTIFACT_TYPES;
    public flatMapOfSteps: {id: string, isRunning: boolean}[] = [];

    private getArtifactsSubscription: Subscription;
    private allSelected: boolean = false;
    private selectedCount: number = 0;

    constructor(private artifactsService: ArtifactsService, private launchPanelService: LaunchPanelService) {
    }

    public ngOnDestroy() {
        this.artifactSubscriptionsCleanup();
    }

    public ngOnChanges(changes: SimpleChanges) {
        if (changes.task) {
            let val = changes.task.currentValue;
            if (val && val.id) {
                if (val.id === val.task_id) {
                    this.loadArtifacts(val.id, null);
                } else {
                    this.loadArtifacts(null, val.id);
                }
                this.flatMapOfSteps = JobTreeNode.createFromTask(val).getFlattenNodes().map(item => ({
                    id: item.value.id,
                    isRunning: item.value.status === TaskStatus.Running,
                }));
            }
        }
    }

    private loadArtifacts(workflowId: string, serviceInstanceId: string): void {
        this.artifactGroups = [
            {
                name: ARTIFACT_TYPES.EXPORTED,
                artifacts: [],
            },
            {
                name: ARTIFACT_TYPES.INTERNAL,
                artifacts: [],
            },
            {
                name: ARTIFACT_TYPES.USER_LOG,
                artifacts: [],
            },
            {
                name: ARTIFACT_TYPES.AX_LOG_EXTERNAL,
                artifacts: [],
            },
        ];
        this.dataLoaded = false;
        this.isArtifactGroupsEmpty = true;
        this.selectedCount = 0;
        this.allSelected = false;
        // the artifactSubscriptionsCleanup() is required. After implementing SSE in job-details, the list of artifacts
        // is refreshed for each event, sometimes there are 2 events send in the same time and artifact items are doubled
        this.artifactSubscriptionsCleanup();
        let params = {
            artifact_type: this.artifactTypeFilter.join(),
        };
        if (workflowId) {
            params['workflow_id'] = workflowId;
        } else if (serviceInstanceId) {
            params['service_instance_id'] = serviceInstanceId;
        }
        this.getArtifactsSubscription = this.artifactsService.getArtifacts(params, true).subscribe(success => {
            let allArtifacts = success;

            this.isArtifactGroupsEmpty = !allArtifacts.length;

            SortOperations.sortBy(<SelectableArtifact[]>allArtifacts, 'artifact_type')
                .map((artifact: SelectableArtifact, i: number, artifacts: SelectableArtifact[]) => {
                    if (artifact.source_artifact_id !== '') {
                        artifact.alias_of = artifacts.find((sourceArtifact: SelectableArtifact) =>
                            artifact.source_artifact_id === sourceArtifact.artifact_id
                        ).alias;
                    }
                    return artifact;
                }).forEach((artifact: SelectableArtifact) => {
                    for (let i = 0; i < this.artifactGroups.length; i++) {
                        let artifactType = artifact.artifact_type === ARTIFACT_TYPES.AX_LOG ? ARTIFACT_TYPES.AX_LOG_EXTERNAL : artifact.artifact_type;
                        if (this.artifactGroups[i].name === artifactType) {
                            this.artifactGroups[i].artifacts.push(artifact);
                            return;
                        }
                    }
                });

            this.artifactGroups.forEach(artifactGroup => {
                SortOperations.sortBy(artifactGroup.artifacts, 'name');
            });
            this.dataLoaded = true;
        }, () => {
            this.dataLoaded = true;
        });
    }

    public selectArtifact(artifact: SelectableArtifact): void {
        this.allSelected = false;
        artifact.selected = !artifact.selected;
        artifact.selected ? this.selectedCount++ : this.selectedCount--;
        this.selectedCountChanged.emit(this.selectedCount);
    }

    public selectAllArtifacts(): void {
        this.allSelected = !this.allSelected;
        this.selectedCount = 0;

        this.artifactGroups.forEach((group: ArtifactGroup) => {
            group.artifacts.forEach((artifact: SelectableArtifact) => {
                artifact.selected = this.allSelected;

                if (this.allSelected) {
                    this.selectedCount++;
                }
            });
        });
        this.selectedCountChanged.emit(this.selectedCount);
    }

    public downloadArtifact(artifact: SelectableArtifact, openInWindow?: boolean): void {
        let address = this.getArtifactDownloadUrl(artifact.artifact_id);
        openInWindow ? window.open(address) : window.location.href = address;
    }

    public downloadSelectedArtifacts(): void {
        // fallback to windows.open if download attr not supported
        if (typeof document.createElement('a').download === 'undefined') {
            this.artifactGroups.forEach((group: ArtifactGroup) => {
                group.artifacts.forEach((artifact: SelectableArtifact) => {
                    if (artifact.selected) {
                        this.downloadArtifact(artifact, this.selectedCount > 1);
                    }
                });
            });
        } else {
            this.aElements.forEach((aElement: ElementRef) => {
                if (aElement.nativeElement.getAttribute('data-selected') === 'true') {
                    aElement.nativeElement.dispatchEvent(new MouseEvent('click'));
                }
            });
        }
    }

    public launchWorkflowForExportedArtifact(artifactGroup: ArtifactGroup) {
        let commit;
        commit = this.task.commit;
        // GUI-1829 this is required for jobs comes not from commit
        if (!commit.hasOwnProperty('branch') && !commit.hasOwnProperty('repo') && !commit.hasOwnProperty('revision')) {
            commit['repo'] = this.task.arguments.hasOwnProperty('repo') ? this.task.arguments['repo'] : null;
            commit['branch'] = this.task.arguments.hasOwnProperty('commit') ? this.task.arguments['commit'] : null;
            commit['revision'] = this.task.hasOwnProperty('id') ? this.task['id'] : null;
        }
        this.launchPanelService.openPanel(commit, null, false, artifactGroup.artifacts);
    }

    public isDeleted(deletedStatus): boolean {
        return (deletedStatus === DeletedStatus.TemporaryDeleted || deletedStatus === DeletedStatus.PermanentlyDeleted);
    }

    public isRunning(instanceId: string) {
        let step = this.flatMapOfSteps.find(s => s.id === instanceId);
        return step ? step.isRunning : false;
    }

    public ifAnyIsRunning () {
        let step = this.flatMapOfSteps.find(s => s.isRunning);
        return step ? step.isRunning : false;
    }

    private getArtifactDownloadUrl(artifactId: string): string {
        let filter = new URLSearchParams();
        filter.set('action', 'download');
        filter.append('artifact_id', artifactId);
        return `v1/artifacts?${filter.toString()}`;
    }

    private artifactSubscriptionsCleanup() {
        if (this.getArtifactsSubscription) {
            this.getArtifactsSubscription.unsubscribe();
        }
    }
}
