import { Component, Input } from '@angular/core';

import { Project } from '../../../model';

@Component({
    selector: 'ax-project-details-panel',
    templateUrl: './project-details-panel.html',
    styles:  [ require('./project-details-panel.scss') ],
})
export class ProjectDetailsPanelComponent {

    @Input()
    public project: Project;

    get pageTitle(): string {
        return this.project && this.project.name;
    }

    public get tags(): string[] {
        return this.project && this.project.labels && this.project.labels.tags || [];
    }

    public byFieldName(i, field) {
        return field.name;
    }

    public get customFields(): {name: string, value: string }[] {
        let labels = this.project && this.project.labels || {};
        return Object.keys(labels).filter(key => key !== 'tags').map(key => {
            return {
                name: key,
                value: labels[key]
            };
        });
    }
}
