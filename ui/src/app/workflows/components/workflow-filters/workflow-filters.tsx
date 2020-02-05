import * as React from 'react';
import {TagsInput} from '../../../shared/components/tags-input/tags-input';
import {NamespaceFilter} from '../../../shared/components/namespace-filter';
import {CheckboxFilter} from '../../../shared/components/checkbox-filter/checkbox-filter';
import * as models from '../../../../models';

require('./workflow-filters.scss');

function getLabelsSuggestions(labels: Map<string, Set<string>>) {
    const suggestions = new Array<string>();
    Array.from(labels.entries()).forEach(([label, values]) => {
        values.forEach(val => suggestions.push(`${label}=${val}`));
    });
    return suggestions;
}

interface WorkflowFilterProps {
    workflows: models.Workflow[];
    namespace: string;
    phases: string[];
    labels: string[];
    onChange: (namespace: string, phases: string[], labels: string[]) => void;
}

interface WorkflowFilterState {
    namespace: string;
    phases: string[];
    labels: string[];
    error?: Error;
}

export class WorkflowFilters extends React.Component<WorkflowFilterProps, WorkflowFilterState> {
    constructor(props: Readonly<WorkflowFilterProps>) {
        super(props);
        this.state = {
            namespace: props.namespace,
            phases: props.phases,
            labels: props.labels
        };
    }

    private set namespace(namespace: string) {
        this.setState({namespace});
    }

    private get namespace() {
        return this.state.namespace;
    }

    private set phases(phases: string[]) {
        this.setState({phases});
    }

    private get phases() {
        return this.state.phases;
    }

    private set labels(labels: string[]) {
        this.setState({labels});
    }

    private get labels() {
        return this.state.labels;
    }

    render() {
        const {workflows, onChange} = this.props;
        const phasesMap = new Map<string, number>();
        Object.values(models.NODE_PHASE).forEach(value => phasesMap.set(value, 0));
        workflows.filter(wf => wf.status.phase).forEach(wf => phasesMap.set(wf.status.phase, (phasesMap.get(wf.status.phase) || 0) + 1));

        const labelsMap = new Map<string, Set<string>>();
        workflows
            .filter(wf => wf.metadata && wf.metadata.labels)
            .forEach(wf =>
                Object.keys(wf.metadata.labels).forEach(label => {
                    let values = labelsMap.get(label);
                    if (!values) {
                        values = new Set<string>();
                        labelsMap.set(label, values);
                    }
                    values.add(wf.metadata.labels[label]);
                })
            );
        return (
            <div className='wf__filters-container'>
                <div className='columns small-12 medium-3 xxlarge-12'>
                    <div className='row'>
                        <p className='wf__filters-container-title'>Namespace</p>
                        <NamespaceFilter
                            value={this.namespace}
                            onChange={ns => {
                                this.namespace = ns;
                                onChange(ns, this.state.phases, this.state.labels);
                            }}
                        />
                    </div>
                    <div className='row'>
                        <p className='wf__filters-container-title'>Phases</p>
                        <CheckboxFilter
                            selected={this.phases}
                            onChange={selected => {
                                this.phases = selected;
                                onChange(this.namespace, selected, this.labels);
                            }}
                            items={Array.from(phasesMap.keys()).map(phase => ({name: phase, count: phasesMap.get(phase) || 0}))}
                            type='phase'
                        />
                    </div>
                    <div className='row'>
                        <p className='wf__filters-container-title'>Labels</p>
                        <TagsInput
                            placeholder=''
                            autocomplete={getLabelsSuggestions(labelsMap)}
                            tags={this.labels}
                            onChange={tags => {
                                this.labels = tags;
                                onChange(this.namespace, this.phases, tags);
                            }}
                        />
                    </div>
                </div>
            </div>
        );
    }
}
