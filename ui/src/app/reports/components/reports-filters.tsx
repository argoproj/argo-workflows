import * as React from 'react';

import {NODE_PHASE} from '../../../models';
import {DataLoaderDropdown} from '../../shared/components/data-loader-dropdown';
import {NamespaceFilter} from '../../shared/components/namespace-filter';
import {TagsInput} from '../../shared/components/tags-input/tags-input';
import {services} from '../../shared/services';

import './reports-filters.scss';

const labelKeyPhase = 'workflows.argoproj.io/phase';
const labelKeyWorkflowTemplate = 'workflows.argoproj.io/workflow-template';
const labelKeyCronWorkflow = 'workflows.argoproj.io/cron-workflow';

interface ReportFiltersProps {
    namespace: string;
    labels: string[];
    onChange: (newNamespace: string, newLabels: string[]) => void;
}

export function ReportFilters({namespace, labels, onChange}: ReportFiltersProps) {
    function getLabel(name: string) {
        return (labels.find(label => label.startsWith(name)) || '').replace(name + '=', '');
    }

    function setLabel(name: string, value: string) {
        onChange(namespace, labels.filter(label => !label.startsWith(name)).concat(name + '=' + value));
    }

    function getPhase() {
        return getLabel(labelKeyPhase);
    }

    function setPhase(value: string) {
        setLabel(labelKeyPhase, value);
    }

    function setWorkflowTemplate(value: string) {
        setLabel(labelKeyWorkflowTemplate, value);
    }

    function setCronWorkflow(value: string) {
        setLabel(labelKeyCronWorkflow, value);
    }

    return (
        <div className='wf-filters-container'>
            <div className='row'>
                <div className=' columns small-12 xlarge-12'>
                    <p className='wf-filters-container__title'>Namespace</p>
                    <NamespaceFilter
                        value={namespace}
                        onChange={newNamespace => {
                            onChange(newNamespace, labels);
                        }}
                    />
                </div>
                <div className=' columns small-12 xlarge-12'>
                    <p className='wf-filters-container__title'>Labels</p>
                    <TagsInput placeholder='Labels' tags={labels} onChange={newLabels => onChange(namespace, newLabels)} />
                </div>
                <div className=' columns small-12 xlarge-12'>
                    <p className='wf-filters-container__title'>Workflow Template</p>
                    <DataLoaderDropdown
                        load={async () => {
                            const list = await services.workflowTemplate.list(namespace, []);
                            return (list.items || []).map(x => x.metadata.name);
                        }}
                        onChange={value => setWorkflowTemplate(value)}
                    />
                </div>
                <div className=' columns small-12 xlarge-12'>
                    <p className='wf-filters-container__title'>Cron Workflow</p>
                    <DataLoaderDropdown
                        load={async () => {
                            const list = await services.cronWorkflows.list(namespace);
                            return list.map(x => x.metadata.name);
                        }}
                        onChange={value => setCronWorkflow(value)}
                    />
                </div>
                <div className=' columns small-12 xlarge-12'>
                    <p className='wf-filters-container__title'>Phase</p>
                    {[NODE_PHASE.SUCCEEDED, NODE_PHASE.ERROR, NODE_PHASE.FAILED].map(phase => (
                        <div key={phase}>
                            <label style={{marginRight: 10}}>
                                <input type='radio' checked={phase === getPhase()} onChange={() => setPhase(phase)} style={{marginRight: 5}} />
                                {phase}
                            </label>
                        </div>
                    ))}
                </div>
            </div>
        </div>
    );
}
