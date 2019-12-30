import * as classNames from 'classnames';
import * as React from 'react';

import * as models from '../../../../models';
import {Utils} from '../../../shared/utils';

import {WorkflowSteps} from '../workflow-steps/workflow-steps';

require('./workflow-list-item.scss');

export interface WorkflowListItemProps {
    workflow: models.Workflow;
}

export const WorkflowListItem = (props: WorkflowListItemProps) => (
    <div className='workflow-list-item'>
        <div className='workflow-list-item__top'>
            <div className='workflow-list-item__status'>
                <div className='workflow-list-item__status-icon'>
                    <i className={classNames('fa', Utils.statusIconClasses(props.workflow.status.phase))} aria-hidden='true' />
                </div>
                <div className='workflow-list-item__status-message'>{props.workflow.metadata.creationTimestamp}</div>
            </div>
        </div>

        <div className='workflow-list-item__content'>
            <div className='row collapse'>
                <div className='columns medium-7'>
                    <div className='workflow-list-item__content-box'>
                        <WorkflowSteps workflow={props.workflow} />
                    </div>
                </div>
                <div className='columns medium-5'>
                    <div className='workflow-list-item__content-details'>
                        <div className='workflow-list-item__content-details-row row'>
                            <div className='columns large-4'>NAME:</div>
                            <div className='columns large-8'>{props.workflow.metadata.name}</div>
                        </div>
                        <div className='workflow-list-item__content-details-row row'>
                            <div className='columns large-4'>NAMESPACE:</div>
                            <div className='columns large-8'>{props.workflow.metadata.namespace}</div>
                        </div>
                        <div className='workflow-list-item__content-details-row row'>
                            <div className='columns large-4'>CREATED AT:</div>
                            <div className='columns large-8'>{props.workflow.metadata.creationTimestamp}</div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
);
