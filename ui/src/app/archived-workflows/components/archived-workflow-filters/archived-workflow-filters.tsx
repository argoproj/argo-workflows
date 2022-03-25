import * as React from 'react';
import DatePicker from 'react-datepicker';
import * as models from '../../../../models';
import {CheckboxFilter} from '../../../shared/components/checkbox-filter/checkbox-filter';
import {InputFilter} from '../../../shared/components/input-filter';
import {NamespaceFilter} from '../../../shared/components/namespace-filter';
import {TagsInput} from '../../../shared/components/tags-input/tags-input';
import {services} from '../../../shared/services';

import 'react-datepicker/dist/react-datepicker.css';

require('./archived-workflow-filters.scss');

interface ArchivedWorkflowFilterProps {
    workflows: models.Workflow[];
    namespace: string;
    name: string;
    namePrefix: string;
    phaseItems: string[];
    selectedPhases: string[];
    selectedLabels: string[];
    minStartedAt?: Date;
    maxStartedAt?: Date;
    onChange: (namespace: string, name: string, namePrefix: string, selectedPhases: string[], labels: string[], minStartedAt: Date, maxStartedAt: Date) => void;
}

interface State {
    labels: string[];
}

export class ArchivedWorkflowFilters extends React.Component<ArchivedWorkflowFilterProps, State> {
    constructor(props: ArchivedWorkflowFilterProps) {
        super(props);
        this.state = {
            labels: []
        };
    }

    public componentDidMount(): void {
        this.fetchArchivedWorkflowsLabelKeys();
    }

    public render() {
        return (
            <div className='wf-filters-container'>
                <div className='row'>
                    <div className='columns small-2 xlarge-12'>
                        <p className='wf-filters-container__title'>Namespace</p>
                        <NamespaceFilter
                            value={this.props.namespace}
                            onChange={ns => {
                                this.props.onChange(
                                    ns,
                                    this.props.name,
                                    this.props.namePrefix,
                                    this.props.selectedPhases,
                                    this.props.selectedLabels,
                                    this.props.minStartedAt,
                                    this.props.maxStartedAt
                                );
                            }}
                        />
                    </div>
                    <div className='columns small-2 xlarge-12'>
                        <p className='wf-filters-container__title'>Name</p>
                        <InputFilter
                            value={this.props.name}
                            name='wfname'
                            onChange={wfname => {
                                this.props.onChange(
                                    this.props.namespace,
                                    wfname,
                                    this.props.namePrefix,
                                    this.props.selectedPhases,
                                    this.props.selectedLabels,
                                    this.props.minStartedAt,
                                    this.props.maxStartedAt
                                );
                            }}
                        />
                    </div>
                    <div className='columns small-2 xlarge-12'>
                        <p className='wf-filters-container__title'>Name Prefix</p>
                        <InputFilter
                            value={this.props.namePrefix}
                            name='wfnamePrefix'
                            onChange={wfnamePrefix => {
                                this.props.onChange(
                                    this.props.namespace,
                                    this.props.name,
                                    wfnamePrefix,
                                    this.props.selectedPhases,
                                    this.props.selectedLabels,
                                    this.props.minStartedAt,
                                    this.props.maxStartedAt
                                );
                            }}
                        />
                    </div>
                    <div className='columns small-2 xlarge-12'>
                        <p className='wf-filters-container__title'>Labels</p>
                        <TagsInput
                            placeholder=''
                            autocomplete={this.state.labels}
                            sublistQuery={this.fetchArchivedWorkflowsLabels}
                            tags={this.props.selectedLabels}
                            onChange={tags => {
                                this.props.onChange(
                                    this.props.namespace,
                                    this.props.name,
                                    this.props.namePrefix,
                                    this.props.selectedPhases,
                                    tags,
                                    this.props.minStartedAt,
                                    this.props.maxStartedAt
                                );
                            }}
                        />
                    </div>
                    <div className='columns small-3 xlarge-12'>
                        <p className='wf-filters-container__title'>Phases</p>
                        <CheckboxFilter
                            selected={this.props.selectedPhases}
                            onChange={selected => {
                                this.props.onChange(
                                    this.props.namespace,
                                    this.props.name,
                                    this.props.namePrefix,
                                    selected,
                                    this.props.selectedLabels,
                                    this.props.minStartedAt,
                                    this.props.maxStartedAt
                                );
                            }}
                            items={this.getPhaseItems(this.props.workflows)}
                            type='phase'
                        />
                    </div>
                    <div className='columns small-5 xlarge-12'>
                        <p className='wf-filters-container__title'>Started Time</p>
                        <DatePicker
                            selected={this.props.minStartedAt}
                            onChange={date => {
                                this.props.onChange(
                                    this.props.namespace,
                                    this.props.name,
                                    this.props.namePrefix,
                                    this.props.selectedPhases,
                                    this.props.selectedLabels,
                                    date,
                                    this.props.maxStartedAt
                                );
                            }}
                            placeholderText='From'
                            dateFormat='dd MMM yyyy'
                            todayButton='Today'
                            className='argo-field argo-textarea'
                        />
                        <DatePicker
                            selected={this.props.maxStartedAt}
                            onChange={date => {
                                this.props.onChange(
                                    this.props.namespace,
                                    this.props.name,
                                    this.props.namePrefix,
                                    this.props.selectedPhases,
                                    this.props.selectedLabels,
                                    this.props.minStartedAt,
                                    date
                                );
                            }}
                            placeholderText='To'
                            dateFormat='dd MMM yyyy'
                            todayButton='Today'
                            className='argo-field argo-textarea'
                        />
                    </div>
                </div>
            </div>
        );
    }

    private getPhaseItems(workflows: models.Workflow[]) {
        const phasesMap = new Map<string, number>();
        this.props.phaseItems.forEach(value => phasesMap.set(value, 0));
        workflows.filter(wf => wf.status.phase).forEach(wf => phasesMap.set(wf.status.phase, (phasesMap.get(wf.status.phase) || 0) + 1));
        const results = new Array<{name: string; count: number}>();
        phasesMap.forEach((val, key) => {
            results.push({name: key, count: val});
        });
        return results;
    }

    private fetchArchivedWorkflowsLabelKeys(): void {
        services.archivedWorkflows.listLabelKeys().then(list => {
            this.setState({
                labels: list.items.sort((a, b) => a.localeCompare(b)) || []
            });
        });
    }

    private fetchArchivedWorkflowsLabels(key: string): Promise<any> {
        return services.archivedWorkflows.listLabelValues(key).then(list => {
            return list.items.map(i => key + '=' + i).sort((a, b) => a.localeCompare(b));
        });
    }
}
