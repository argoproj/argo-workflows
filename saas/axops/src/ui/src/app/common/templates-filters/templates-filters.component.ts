import { Component, Input,  } from '@angular/core';
import * as _ from 'lodash';

import { TemplateService } from '../../services';
import { Template } from '../../model';
import { SortOperations } from '../../common';

@Component({
    selector: 'ax-templates-filters',
    templateUrl: './templates-filters.html',
    styles: [ require('./templates-filters.scss') ],
})
export class TemplatesFiltersComponent {

    private tepmlateNamesToSelect: string[];
    private selectedTemplates: Template[] = [];
    public templates: { name: string, value: string, selected: boolean, visible: boolean }[] = [];
    public loading: boolean = false;
    public filterString: string = '';

    @Input()
    set selectedTemplateNames(value: string) {
        if (value) {
            this.tepmlateNamesToSelect = value.split(',');
        } else {
            this.selectedTemplates = [];
            this.tepmlateNamesToSelect = [];
            this.updateSelectedValues();
        }
    }

    get selectedTemplateNames(): string {
        return this.selectedTemplates.map(t => t.name).join(',');
    }

    constructor(private templateService: TemplateService) {
    }

    public loadTemplates() {
        this.loading = true;

        this.templateService.getTemplatesAsync({ dedup: true, fields: ['id', 'name'], sort: 'name'}, false).toPromise().then((res: { data: Template[]}) => {
            let templatesList = res.data.map(template => {
                return { name: template.name, value: template.name, selected: false, visible: true };
            });

            this.templates =
                SortOperations.sortBy(templatesList, 'name', true).filter((value, index, array) => (index === 0) || (value.name !== array[index - 1].name));

            this.loading = false;
            this.updateSelectedValues();
        }, error => {
            this.templates = [];
        });
    }

    public selectTemplate(template) {
        template.selected = !template.selected;
        if (template.selected) {
            this.selectedTemplates.push(template);
        } else {
            _.remove(this.selectedTemplates, (t: Template) => t.name === template.name);
        }

        this.selectedTemplateNames = this.selectedTemplates.map(t => t.name).join(',');
    }

    public filter(input: string) {
        if (this.filterString.length > 0) {
            this.templates.forEach(template => {
                template.visible = template.name.toLowerCase().indexOf(this.filterString.toLowerCase()) !== -1;
            });
        } else {
            this.templates.forEach(template => template.visible = true);
        }
    }

    private updateSelectedValues() {
        if (this.tepmlateNamesToSelect.length) {
            this.tepmlateNamesToSelect.forEach(templateName => {
                let templateIndex = _.findIndex(this.templates, (template) => template.name === templateName);
                if (templateIndex !== -1) {
                    this.templates[templateIndex].selected = true;
                }
            });
        } else {
            this.tepmlateNamesToSelect = [];
            this.templates.forEach(template => template.selected = false);
        }
    }
}
