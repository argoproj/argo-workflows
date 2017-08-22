import { Component, OnInit, Input } from '@angular/core';

import { LabelService } from '../../services';
import { Label } from '../../model';

@Component({
    selector: 'ax-labels-filters',
    templateUrl: './labels-filters.html',
    providers: [LabelService],
    styles: [ require('./labels-filters.scss') ],
})
export class LabelsFiltersComponent implements OnInit {

    @Input() set setSelectedLabels(value: string) {
        this.selectedLabels = value;
        this.updateSelectedValues();
    }

    @Input()
    labelsType: string = null;

    selectedLabels: string = null;
    labelGroups: { labelKey: string, labels: Label[] }[] = [];

    constructor(private labelService: LabelService) { }

    ngOnInit() {
        this.labelService.getLabels({ type: this.labelsType }).subscribe(
            success => {
                this.labelGroups = this.labelService.groupLabelsByKey(<Label[]>success['data']);
                this.updateSelectedValues();
            }
        );
    }

    updateSelectedValues() {
        let selectedGroups = this.selectedLabels ? this.selectedLabels.split(';') : [];

        this.labelGroups.forEach(g => {
            let group = selectedGroups.filter(sg => { return sg.indexOf(`${g.labelKey}:`) >= 0; })[0];
            if (group) {
                g.labels.forEach(l => {
                    l.selected = group.indexOf(l.value) >= 0;
                });
            } else {
                g.labels.forEach(l => { l.selected = false; });
            }
        });
    }

    selectLabel(label) {
        label.selected = !(label.selected);
        let selectedLabels = [];
        this.labelGroups.forEach(g => {
            let sl = g.labels.filter(l => {
                return l.selected;
            }).map(l => {
                return l.value;
            }).join(',');
            if (sl.length > 0) {
                selectedLabels.push(`${g.labelKey}:${sl}`);
            }
        });
        this.selectedLabels = selectedLabels.join(';');
    }
}
