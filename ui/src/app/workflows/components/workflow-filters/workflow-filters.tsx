import * as React from 'react';
import {useMemo} from 'react';
import DatePicker from 'react-datepicker';
import 'react-datepicker/dist/react-datepicker.css';

import * as models from '../../../../models';
import {WorkflowPhase} from '../../../../models';
import {CheckboxFilter} from '../../../shared/components/checkbox-filter/checkbox-filter';
import {DataLoaderDropdown} from '../../../shared/components/data-loader-dropdown';
import {NamespaceFilter} from '../../../shared/components/namespace-filter';
import {TagsInput} from '../../../shared/components/tags-input/tags-input';
import {services} from '../../../shared/services';

import {InputFilter} from '../../../shared/components/input-filter';
import './workflow-filters.scss';
import {DropDown} from '../../../shared/components/dropdown/dropdown';
import classNames from 'classnames';

interface WorkflowFilterProps {
    workflows: models.Workflow[];
    namespace: string;
    phaseItems: WorkflowPhase[];
    phases: WorkflowPhase[];
    labels: string[];
    createdAfter?: Date;
    finishedBefore?: Date;
    setNamespace: (namespace: string) => void;
    setPhases: (phases: WorkflowPhase[]) => void;
    setLabels: (labels: string[]) => void;
    setCreatedAfter: (createdAfter: Date) => void;
    setFinishedBefore: (finishedBefore: Date) => void;
    name: string;
    setName: (name: string) => void;
    namePrefix: string;
    setNamePrefix: (namePrefix: string) => void;
    namePattern: string;
    setNamePattern: (namePattern: string) => void;
}

const NAME_FILTERS = [
    {
        title: 'Name Pattern',
        id: 'namePattern'
    },
    {
        title: 'Name Prefix',
        id: 'namePrefix'
    },
    {
        title: 'Name Exact',
        id: 'name'
    }
];

export function WorkflowFilters(props: WorkflowFilterProps) {
    const [nameFilter, setNameFilter] = React.useState(() => {
        if (props.namePrefix) {
            return NAME_FILTERS[1];
        }
        if (props.name) {
            return NAME_FILTERS[2];
        }
        return NAME_FILTERS[0];
    });
    function setLabel(name: string, value: string) {
        props.setLabels([name.concat('=' + value)]);
    }

    function setWorkflowTemplate(value: string) {
        setLabel(models.labels.workflowTemplate, value);
    }

    function setCronWorkflow(value: string) {
        setLabel(models.labels.cronWorkflow, value);
    }

    const labelSuggestion = useMemo(() => {
        return getLabelSuggestions(props.workflows);
    }, [props.workflows]);

    const phaseItems = useMemo(() => {
        const phasesMap = new Map<string, number>();
        props.phaseItems.forEach(value => phasesMap.set(value, 0));
        props.workflows.filter(wf => wf.status.phase).forEach(wf => phasesMap.set(wf.status.phase, (phasesMap.get(wf.status.phase) || 0) + 1));

        const results = new Array<{name: string; count: number}>();
        phasesMap.forEach((val, key) => {
            results.push({name: key, count: val});
        });
        return results;
    }, [props.workflows, props.phaseItems]);

    const handleNameFilterChange = (item: {title: string; id: string}) => {
        props.setNamePrefix('');
        props.setNamePattern('');
        props.setName('');
        setNameFilter(item);
    };

    return (
        <div className='wf-filters-container'>
            <div className='row'>
                <div className='columns small-2 xlarge-12'>
                    <p className='wf-filters-container__title'>Namespace</p>
                    <NamespaceFilter value={props.namespace} onChange={props.setNamespace} />
                </div>
                <div className='columns small-2 xlarge-12'>
                    <DropDown
                        isMenu
                        closeOnInsideClick
                        anchor={
                            <div className={classNames('top-bar__filter')} title='Filter'>
                                <p className='wf-filters-container__title'>
                                    {nameFilter.title} <i className='fa fa-angle-down' aria-hidden='true' />
                                </p>
                            </div>
                        }>
                        <ul id='top-bar__filter-list'>
                            {NAME_FILTERS.map((item, i) => (
                                <li key={i} className={classNames('top-bar__filter-item', {title: true})} onClick={() => handleNameFilterChange(item)}>
                                    <span>{item.title}</span>
                                </li>
                            ))}
                        </ul>
                    </DropDown>
                    {nameFilter.id === 'namePrefix' ? <InputFilter value={props.namePrefix} name='wfNamePrefix' onChange={props.setNamePrefix} placeholder='Search...' /> : null}
                    {nameFilter.id === 'namePattern' ? (
                        <InputFilter value={props.namePattern} name='wfNamePattern' onChange={props.setNamePattern} placeholder='Search...' />
                    ) : null}
                    {nameFilter.id === 'name' ? <InputFilter value={props.name} name='wfName' onChange={props.setName} placeholder='Search...' /> : null}
                </div>
                <div className='columns small-2 xlarge-12'>
                    <p className='wf-filters-container__title'>Labels</p>
                    <TagsInput placeholder='' autocomplete={labelSuggestion} tags={props.labels} onChange={props.setLabels} />
                </div>
                <div className='columns small-2 xlarge-12'>
                    <p className='wf-filters-container__title'>Workflow Template</p>
                    <DataLoaderDropdown
                        load={async () => {
                            const list = await services.workflowTemplate.list(props.namespace, []);
                            return (list.items || []).map(x => x.metadata.name);
                        }}
                        onChange={setWorkflowTemplate}
                    />
                </div>
                <div className='columns small-2 xlarge-12'>
                    <p className='wf-filters-container__title'>Cron Workflow</p>
                    <DataLoaderDropdown
                        load={async () => {
                            const list = await services.cronWorkflows.list(props.namespace);
                            return list.map(x => x.metadata.name);
                        }}
                        onChange={setCronWorkflow}
                    />
                </div>
                <div className='columns small-4 xlarge-12'>
                    <p className='wf-filters-container__title'>Phases</p>
                    <CheckboxFilter selected={props.phases} onChange={props.setPhases} items={phaseItems} type='phase' />
                </div>
                <div className='columns small-5 xlarge-12'>
                    <p className='wf-filters-container__title'>Created Since</p>
                    <div className='wf-filters-container__content'>
                        <DatePicker
                            selected={props.createdAfter}
                            onChange={props.setCreatedAfter}
                            placeholderText='From'
                            dateFormat='dd MMM yyyy'
                            todayButton='Today'
                            className='argo-field argo-textarea'
                        />
                        <a onClick={() => props.setCreatedAfter(undefined)}>
                            <i className='fa fa-times-circle' />
                        </a>
                    </div>
                    <p className='wf-filters-container__title'>Finished Before</p>
                    <div className='wf-filters-container__content'>
                        <DatePicker
                            selected={props.finishedBefore}
                            onChange={props.setFinishedBefore}
                            placeholderText='To'
                            dateFormat='dd MMM yyyy'
                            todayButton='Today'
                            className='argo-field argo-textarea'
                        />
                        <a onClick={() => props.setFinishedBefore(undefined)}>
                            <i className='fa fa-times-circle' />
                        </a>
                    </div>
                </div>
            </div>
        </div>
    );
}

function addCommonLabel(suggestions: string[]) {
    const commonLabel = new Array<string>();
    const commonLabelPool = [models.labels.cronWorkflow, models.labels.workflowTemplate, models.labels.clusterWorkflowTemplate];
    commonLabelPool.forEach(labelPrefix => {
        for (const label of suggestions) {
            if (label.startsWith(labelPrefix)) {
                commonLabel.push(`${labelPrefix}`);
                break;
            }
        }
    });
    return commonLabel.concat(suggestions);
}

function getLabelSuggestions(workflows: models.Workflow[]) {
    const suggestions = new Array<string>();
    workflows
        .filter(wf => wf.metadata.labels)
        .forEach(wf => {
            Object.keys(wf.metadata.labels).forEach(label => {
                const value = wf.metadata.labels[label];
                const suggestedLabel = `${label}=${value}`;
                if (!suggestions.some(v => v === suggestedLabel)) {
                    suggestions.push(`${label}=${value}`);
                }
            });
        });
    return addCommonLabel(suggestions.sort((a, b) => a.localeCompare(b)));
}
