import { Component, Input } from '@angular/core';

import { Template } from '../../model';
import { NodeInfo } from '../workflow-tree/workflow-tree.view-models';

@Component({
    selector: 'ax-template-viewer',
    templateUrl: './template-viewer.html',
    styles: [ require('./template-viewer.scss') ],
})
export class TemplateViewerComponent {

    @Input()
    public template: Template;

    @Input()
    public isHiddenYamlBtn: false;

    public selectedStep: string;
    public isYamlVisible: boolean;

    private regexToIcon = new Map<string, string>();

    constructor() {
        this.regexToIcon.set('.*test.*', 'ax-icon-test');
        this.regexToIcon.set('.*checkout.*', 'ax-icon-checkout');
        this.regexToIcon.set('.*(deploy|release).*', 'ax-icon-deploy');
        this.regexToIcon.set('.*approv.*', 'ax-icon-approval');
        this.regexToIcon.set('.*build.*', 'ax-icon-build');
    }

    public selectYaml(node: NodeInfo) {
        this.isYamlVisible = true;
        this.selectedStep = node.name;
    }

    public showYaml() {
        this.isYamlVisible = true;
    }

    public closeYaml() {
        this.isYamlVisible = false;
    }

    public getStepIcon(node: NodeInfo) {
        let icons = Array.from(this.regexToIcon.entries()).
            filter(([regex]) => node.name.toLowerCase().match(regex) != null).
            map(([, icon]) => icon);
        return icons.length > 0 ? icons[0] : 'fa-gear';
    }
}
