import {Checkbox} from 'argo-ui/src/components/checkbox';
import {DropDown} from 'argo-ui/src/components/dropdown/dropdown';
import {TopBarFilter} from 'argo-ui/src/index';
import * as classNames from 'classnames';
import * as React from 'react';
import {WorkflowDagRenderOptions} from './workflow-dag';

export class WorkflowDagRenderOptionsPanel extends React.Component<WorkflowDagRenderOptions & {onChange: (changed: WorkflowDagRenderOptions) => void}> {
    private get workflowDagRenderOptions() {
        return this.props as WorkflowDagRenderOptions;
    }

    public render() {
        const filter: TopBarFilter<string> = {
            items: [
                {content: () => <span>Phase</span>},
                {value: 'phase:Pending', label: 'Pending'},
                {value: 'phase:Running', label: 'Running'},
                {value: 'phase:Succeeded', label: 'Succeeded'},
                {value: 'phase:Skipped', label: 'Skipped'},
                {value: 'phase:Failed', label: 'Failed'},
                {value: 'phase:Error', label: 'Error'},
                {content: () => <span>Type</span>},
                {value: 'type:Pod', label: 'Pod'},
                {value: 'type:Steps', label: 'Steps'},
                {value: 'type:DAG', label: 'DAG'},
                {value: 'type:Retry', label: 'Retry'},
                {value: 'type:Skipped', label: 'Skipped'},
                {value: 'type:Suspend', label: 'Suspend'},
                {value: 'type:TaskGroup', label: 'TaskGroup'},
                {value: 'type:StepGroup', label: 'StepGroup'}
            ],
            selectedValues: this.props.nodesToDisplay,
            selectionChanged: items => {
                this.props.onChange({
                    ...this.workflowDagRenderOptions,
                    nodesToDisplay: items
                });
            }
        };
        return (
            <div className='workflow-dag-render-options-panel'>
                <DropDown
                    isMenu={true}
                    anchor={() => (
                        <div className={classNames('top-bar__filter', {'top-bar__filter--selected': filter.selectedValues !== this.props.nodesToDisplay})}>
                            <i className='argo-icon-filter' aria-hidden='true' />
                            <i className='fa fa-angle-down' aria-hidden='true' />
                        </div>
                    )}>
                    <ul>
                        {filter.items.map((item, i) => (
                            <li key={i} className={classNames({'top-bar__filter-item': !item.content})}>
                                {(item.content && item.content(vals => filter.selectionChanged(vals))) || (
                                    <React.Fragment>
                                        <Checkbox
                                            id={`filter__${item.value}`}
                                            checked={filter.selectedValues.includes(item.value)}
                                            onChange={checked => {
                                                const selectedValues = filter.selectedValues.slice();
                                                const index = selectedValues.indexOf(item.value);
                                                if (index > -1 && !checked) {
                                                    selectedValues.splice(index, 1);
                                                } else {
                                                    selectedValues.push(item.value);
                                                }
                                                filter.selectionChanged(selectedValues);
                                            }}
                                        />
                                        <label htmlFor={`filter__${item.value}`}>{item.label}</label>
                                    </React.Fragment>
                                )}
                            </li>
                        ))}
                    </ul>
                </DropDown>
                <a
                    className={classNames({active: this.props.horizontal})}
                    onClick={() =>
                        this.props.onChange({
                            ...this.workflowDagRenderOptions,
                            horizontal: !this.props.horizontal
                        })
                    }
                    title='Horizontal layout'>
                    <i className='fa fa-project-diagram' />
                </a>
                <a
                    onClick={() =>
                        this.props.onChange({
                            ...this.workflowDagRenderOptions,
                            scale: Math.max(1, this.props.scale / 1.5)
                        })
                    }
                    title='Zoom into the timeline'>
                    <i className='fa fa-search-plus' />
                </a>
                <a
                    onClick={() =>
                        this.props.onChange({
                            ...this.workflowDagRenderOptions,
                            scale: this.props.scale * 1.5
                        })
                    }
                    title='Zoom out from the timeline'>
                    <i className='fa fa-search-minus' />
                </a>
                <a
                    onClick={() =>
                        this.props.onChange({
                            ...this.workflowDagRenderOptions,
                            expandNodes: new Set()
                        })
                    }
                    title='Collapse all nodes'>
                    <i className='fa fa-compress' data-fa-transform='rotate-45' />
                </a>
                <a
                    onClick={() =>
                        this.props.onChange({
                            ...this.workflowDagRenderOptions,
                            expandNodes: new Set(['*'])
                        })
                    }
                    title='Expand all nodes'>
                    <i className='fa fa-expand' data-fa-transform='rotate-45' />
                </a>
                <a
                    className={classNames({active: this.props.fastRenderer})}
                    onClick={() =>
                        this.props.onChange({
                            ...this.workflowDagRenderOptions,
                            fastRenderer: !this.props.fastRenderer
                        })
                    }
                    title='Use a faster, but less pretty, renderer to display the workflow'>
                    <i className='fa fa-bolt' />
                </a>
            </div>
        );
    }
}
