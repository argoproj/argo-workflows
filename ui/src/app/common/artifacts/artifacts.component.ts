import { Component, Input, Output, EventEmitter, OnDestroy, SimpleChanges, ViewChildren, QueryList, ElementRef } from '@angular/core';
import { Subscription } from 'rxjs/Subscription';

import * as models from '../../models';
import { WorkflowTree, ArtifactInfo } from '../workflow-tree/workflow-tree.view-models';

class ArtifactGroup {
    name = '';
    artifacts: ArtifactInfo[] = [];
}

@Component({
    selector: 'ax-artifacts',
    templateUrl: './artifacts.html',
    styles: [ require('./artifacts.scss') ],
})
export class ArtifactsComponent implements OnDestroy {

    @Input()
    public isFullWidthContent = false;

    @Input()
    public set workflowTree(workflowTree: WorkflowTree) {
      const artifacts = workflowTree && workflowTree.getArtifacts() || [];
      if (artifacts.length === 0) {
        this.artifactGroups = [];
      } else {
        this.artifactGroups = [{
          name: 'artifacts',
          artifacts,
        }];
      }
    }

    @Input()
    public nodeName: string;

    public artifactGroups: ArtifactGroup[] = [];

    @ViewChildren('a')
    public aElements: QueryList<ElementRef>;

    private getArtifactsSubscription: Subscription;
    private allSelected = false;
    private selectedCount = 0;

    public ngOnDestroy() {
        this.artifactSubscriptionsCleanup();
    }

    private artifactSubscriptionsCleanup() {
        if (this.getArtifactsSubscription) {
            this.getArtifactsSubscription.unsubscribe();
        }
    }
}
