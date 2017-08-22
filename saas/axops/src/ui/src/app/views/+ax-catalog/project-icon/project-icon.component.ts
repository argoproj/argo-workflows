import { Component, Input } from '@angular/core';

import { Project } from '../../../model';

@Component({
    selector: 'ax-project-icon',
    templateUrl: './project-icon.html',
    styles: [ require('./project-icon.scss') ],
})
export class ProjectIconComponent {

    @Input()
    public project: Project;
}
