import { Component, Input } from '@angular/core';

import { Project } from '../../../model';

@Component({
    selector: 'ax-projects-list',
    templateUrl: 'projects-list.html',
    styles: [ require('./projects-list.scss') ],
})
export class ProjectsListComponent {
    @Input()
    public projects: Project[] = [];

    @Input()
    public searchString: string = '';
}
