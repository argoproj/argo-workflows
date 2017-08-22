import { Component, Input, ElementRef } from '@angular/core';
import { Template } from '../../model';
import { DomSanitizer, SafeHtml } from '@angular/platform-browser';
import { ViewUtils } from '../view-utils';

let yaml = require('json2yaml');

@Component({
    selector: 'ax-yaml-viewer',
    template: '<div class="yaml-viewer" [innerHTML]="html"></div>',
    styles: [ require('./yaml-viewer.scss') ],
})
export class YamlViewerComponent {

    public html: SafeHtml;

    constructor(private sanitized: DomSanitizer, private container: ElementRef) {}

    @Input()
    public set input(value: {template: Template, highlightedStep: string}) {
        if (value.template) {
            let idToTemplate = new Map<string, any>();
            this.getTemplatesMap(value.template, idToTemplate);
            let yamlString = Array.from(idToTemplate.values()).map(item => {
                let itemStr = yaml.stringify(item);
                if (value.highlightedStep) {
                    itemStr = this.highlightStep(item, value.highlightedStep, itemStr);
                }
                itemStr = this.addCounterToDisplayedFiles(itemStr);
                return itemStr;
            }).join('\n\n');
            this.html = this.sanitized.bypassSecurityTrustHtml(yamlString);
            this.navigateToSelection();
        } else {
            this.html = this.sanitized.bypassSecurityTrustHtml('');
        }
    }

    private highlightStep(template, highlightedStepName: string, yamlString: string) {
        let firstLineStepToHighlight = null;
        let lastLineStepToHighlight = null;
        (template['steps'] || []).forEach(stepsGroup => {
            let step = stepsGroup[highlightedStepName];
            if (step) {
                firstLineStepToHighlight = step;
                if (step.parameters) {
                    let last = Object.keys(step.parameters)[Object.keys(step.parameters).length - 1];
                    lastLineStepToHighlight = `${last}: "${step.parameters[last].toString()}"`;
                } else {
                    lastLineStepToHighlight = `template: "${step.template}"`;
                }
            }
        });
        if (firstLineStepToHighlight && lastLineStepToHighlight) {
            let newYamlString = '';
            let isLinePartOfStepToHighlight = false;

            yamlString.split('\n').forEach((line: string, index) => {
                if (line.indexOf(`${highlightedStepName}:`) !== -1 || isLinePartOfStepToHighlight) {
                    isLinePartOfStepToHighlight = line.indexOf(lastLineStepToHighlight) === -1;
                    newYamlString = `${newYamlString}<span>${line}</span>\n`;
                } else {
                    newYamlString = `${newYamlString}${line}\n`;
                }
            });

            yamlString = newYamlString;
        }
        return yamlString;
    }

    private navigateToSelection() {
        setTimeout(() => {
            let viewerHighlight = $('li.highlight', this.container.nativeElement).first();
            if (viewerHighlight.length > 0) {
                let parent = $(ViewUtils.scrollParent(viewerHighlight));
                parent.animate({
                    scrollTop: viewerHighlight.offset().top + parent.scrollTop() - window.pageYOffset - parent.height() / 2
                });
            }
        });
    }

    private addCounterToDisplayedFiles(multilineString: string): string {
        let newMultilineStringWithCounters: string[] = ['<ol>'];
        multilineString.split('\n').forEach(item => {
            if (item !== '') {
                if (item.indexOf('<span>') !== -1) {
                    item = `<li class="highlight">${item}</li>`;
                } else {
                    item = `<li>${item}</li>`;
                }
            }
            newMultilineStringWithCounters.push(item);
        });
        newMultilineStringWithCounters.push('</ol>');
        return newMultilineStringWithCounters.join('\n');
    }

    private getTemplatesMap(template: any, idToTemplate: Map<string, any>) {
        let templateId = `${template.repo}_${template.branch}_${template.name}`;
        let templateSteps = template.steps;
        let templateFixtures = template.fixtures;
        template = this.removeSystemInfo(template);
        idToTemplate.set(templateId, template);

        [{name: 'steps', value: templateSteps}, {name: 'fixtures', value: templateFixtures}].forEach(field => {
            if (field.value) {
                template[field.name] = field.value.map(group => {
                    group = Object.assign({}, group);
                    Object.keys(group).forEach(name => {
                        let item = group[name];
                        if (item.template) {
                            this.getTemplatesMap(item.template, idToTemplate);
                            group[name] = {
                                template: item.template.name,
                                flags: item.flags,
                                parameters: item.parameters,
                            };
                        }
                    });
                    return group;
                });
            }
        });
    }

    // Removes system information fields which are added in run time
    private removeSystemInfo(item) {
        if (typeof item !== 'object') {
            return item;
        }
        item = Object.assign({}, item, {
            id: undefined,
            revision: undefined,
            service_id: undefined,
            status: undefined,
            cost: undefined,
            create_time: undefined,
            launch_time: undefined,
            end_time: undefined,
            wait_time: undefined,
            run_time: undefined,
            average_runtime: undefined,
            artifact_nums: undefined,
            artifact_size: undefined,
            artifact_tags: undefined,
            jobs_fail: undefined,
            jobs_success: undefined,
            is_success: undefined,
            is_failed: undefined,
        });
        for (let key in item) {
            if (item.hasOwnProperty(key)) {
                if (Array.isArray(item[key])) {
                    item[key] = item[key].map(this.removeSystemInfo.bind(this));
                } else if (typeof item[key] === 'object') {
                    item[key] = this.removeSystemInfo(item[key]);
                }
            }
        }
        return item;
    }
}
