import { Component, OnInit } from '@angular/core';
import { ActivatedRoute } from '@angular/router';

import { LayoutSettings, HasLayoutSettings } from '../../layout/layout.component';

import { ProjectService } from '../../../services';
import { Project } from '../../../model';

@Component({
    selector: 'ax-project-details',
    template: '<ax-project-details-panel [project]="project"></ax-project-details-panel>',
})
export class ProjectDetailsComponent implements OnInit, HasLayoutSettings {
    public project: Project;

    constructor(private route: ActivatedRoute, private projectService: ProjectService) {
    }

    public ngOnInit() {
        this.route.params.subscribe(params => {
            this.projectService.getProjectById(params['id']).then(project => {
                this.project = project;
            });
        });
    }

    get layoutSettings(): LayoutSettings {
        return {
            pageTitle: 'CATALOG',
            breadcrumb: [
                {
                    title: 'All Apps',
                    routerLink: ['/app/ax-catalog'],
                },
                {
                  title: this.project && this.project.name,
                },
            ]
        };
    }
}
