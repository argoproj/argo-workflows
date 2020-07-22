export function labelSelectorParams(phases?: Array<string>, labels?: Array<string>) {
    let labelSelector = '';
    if (phases && phases.length > 0) {
        labelSelector = `workflows.argoproj.io/phase in (${phases.join(',')})`;
    }
    if (labels && labels.length > 0) {
        if (labelSelector.length > 0) {
            labelSelector += ',';
        }
        labelSelector += labels.join(',');
    }
    return labelSelector;
}
