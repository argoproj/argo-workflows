import { Component, OnInit } from '@angular/core';
import { HasLayoutSettings, LayoutSettings } from '../../layout';
import { FixtureService, ModalService } from '../../../services';
import { FixtureClass, FixtureTemplate } from '../../../model';
import { DropdownMenuSettings } from 'argo-ui-lib/src/components';

interface TemplateGroup {
    enabledClass: FixtureClass & { stats: {available: number, total: number, percentage: number} };
    templates: FixtureTemplate[];
}

@Component({
    selector: 'ax-fixture-classes',
    templateUrl: './fixture-classes.html',
    styles: [ require('./fixture-classes.scss') ],
})
export class FixtureClassesComponent implements HasLayoutSettings, OnInit {

    public templateGroups: TemplateGroup[] = [];
    public reassignTemplates: FixtureTemplate[] = [];
    public selectedTemplateGroup: TemplateGroup;
    public classIdToReassing: string;
    public hasEnabledClasses: boolean;
    public loading: boolean;

    constructor(private fixtureService: FixtureService, private modalService: ModalService) {
    }

    public ngOnInit() {
        this.loadFixtures();
    }

    public get layoutSettings(): LayoutSettings {
        return {
            pageTitle: 'Fixture Classes',
            breadcrumb: [{
                title: 'All Fixtures',
                routerLink: null
            }],
        };
    }

    public async closeAddPanel(info: { selectedTemplateId: string }) {
        this.selectedTemplateGroup = null;
        if (info.selectedTemplateId) {
            if (this.classIdToReassing === 'create') {
                await this.fixtureService.updateFixtureClass(this.classIdToReassing, info.selectedTemplateId);
            } else {
                await this.fixtureService.createFixtureClass(info.selectedTemplateId);
            }
            this.loadFixtures();
        }
        this.classIdToReassing = null;
    }

    public selectTemplateGroup(templateGroup: TemplateGroup) {
        this.selectedTemplateGroup = templateGroup;
    }

    public getGroupMenu(templateGroup: TemplateGroup) {
        if (templateGroup.enabledClass) {
            return new DropdownMenuSettings([{
                title: 'Reassign Template',
                action: () => {
                    this.classIdToReassing = templateGroup.enabledClass.id;
                    this.selectTemplateGroup(templateGroup);
                },
                iconName: 'ax-icon-connect'
            }, {
                title: 'Delete Class',
                action: () => {
                    this.modalService.showModal(
                        'Delete fixture class?', `Are you sure you want to delete fixture class '${templateGroup.enabledClass.name}'?`).subscribe(async confirmed => {
                            if (confirmed) {
                                await this.fixtureService.deleteFixtureClass(templateGroup.enabledClass.id);
                                this.loadFixtures();
                            }
                        });
                },
                iconName: 'ax-icon-stop'
            }]);
        } else {
            return new DropdownMenuSettings([{
                title: 'Enable',
                action: () => this.selectTemplateGroup(templateGroup),
                iconName: 'ax-icon-play-2'
            }]);
        }
    }

    private async loadFixtures() {
        this.loading = true;
        let templates = await this.fixtureService.getFixtureTemplates();
        let nameToTemplates = new Map<string, FixtureTemplate[]>();
        templates.forEach(template => {
            let repoTemplates = nameToTemplates.get(template.name) || [];
            repoTemplates.push(template);
            nameToTemplates.set(template.name, repoTemplates);
        });

        let usageStats = await this.fixtureService.getUsageStats();
        let fixtureClasses = await this.fixtureService.getFixtureClasses();
        this.hasEnabledClasses = fixtureClasses.length > 0;
        let nameToClass = new Map<string, FixtureClass>();
        fixtureClasses.forEach(fixtureClass => {
            nameToClass.set(fixtureClass.name, fixtureClass);
        });
        this.templateGroups = Array.from(nameToTemplates.entries()).map(([name, repoTemplates]) => {
            let enabledClass = nameToClass.get(name);
            if (enabledClass) {
                nameToClass.delete(name);
            }
            let stats = Object.assign(usageStats[`class_name:${name}`] || { available: 0, total: 0 }, { percentage: 0 });
            if (stats.total > 0) {
                stats.percentage = (stats.available / stats.total) * 100;
            }
            return {
                enabledClass: enabledClass ? Object.assign({}, enabledClass, { stats }) : null,
                templates: repoTemplates,
            };
        });
        Array.from(nameToClass.values()).forEach(enabledClass => {
            this.templateGroups.push({
                enabledClass: Object.assign({}, enabledClass, { stats: { available: 0, total: 0, percentage: 0 } }),
                templates: []
            });
        });
        this.reassignTemplates = this.templateGroups.map(item => item.templates).reduce((first, second) => first.concat(second), []);
        this.loading = false;
    }
}
