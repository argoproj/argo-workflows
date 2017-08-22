import { Component, Input, EventEmitter, Output } from '@angular/core';

import { FixtureTemplate } from '../../../model';
import { FixtureService } from '../../../services';

@Component({
    selector: 'ax-fixture-template-panel',
    templateUrl: './fixture-template-panel.html',
    styles: [ require('./fixture-template-panel.scss') ]
})
export class FixtureTemplatePanelComponent {

    public selectedTemplate: FixtureTemplate;
    public templatesByRepo: FixtureTemplate[][] = [];
    public selectedTemplates: FixtureTemplate[];

    @Input()
    public set templateGroup(templates: FixtureTemplate[]) {
        if (templates) {
            let templatesByRepo = new Map<string, FixtureTemplate[]>();
            templates.forEach(template => {
                let repoTemplates = templatesByRepo.get(template.repo) || [];
                repoTemplates.push(template);
                templatesByRepo.set(template.repo, repoTemplates);
            });
            this.templatesByRepo = Array.from(templatesByRepo.values());
        } else {
            this.templatesByRepo = null;
            this.selectTemplates(null);
        }
    }

    @Input()
    public mode: 'create' | 'reassign' = 'create';

    public get messages() {
        switch (this.mode) {
            case 'create':
                return {
                    title: 'Enable Fixture Class'
                };
            case 'reassign':
                return {
                    title: 'Reassign Fixture Template'
                };
        }
    }

    @Output()
    public onClosePanel = new EventEmitter<{ selectedTemplateId: string }>();

    private templateSearchQuery: string = '';

    constructor(private fixtureService: FixtureService) {
    }

    public closePanel(selectedTemplateId = null) {
        this.onClosePanel.emit({ selectedTemplateId });
    }

    public async save() {
        this.closePanel(this.selectedTemplate.id);
    }

    public get selectedGroupTemplates(): {name: string, value: FixtureTemplate}[] {
        let items = this.selectedTemplates && this.selectedTemplates.map(template => ({
            name: template.branch,
            value: template
        })) || [];
        if (this.templateSearchQuery) {
            items = items.filter(item => item.name.toLocaleLowerCase().indexOf(this.templateSearchQuery.toLocaleLowerCase()) > -1);
        }
        return items;
    }

    public searchQuery(query: string) {
        this.templateSearchQuery = query;
    }

    public selectTemplate(item: { name: string, value: FixtureTemplate}) {
        this.selectedTemplate = item.value;
    }

    public selectTemplates(templates: FixtureTemplate[]) {
        this.selectedTemplates = templates;
        this.selectedTemplate = templates && templates[0];
        this.templateSearchQuery = '';
    }
}
