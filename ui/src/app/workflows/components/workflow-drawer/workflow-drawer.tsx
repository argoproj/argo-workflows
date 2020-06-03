import * as React from 'react';
import * as models from '../../../../models';

import {WorkflowLabels} from '../workflow-labels/workflow-labels';

require('./workflow-drawer.scss');

interface WorkflowDrawerProps {
    workflow: models.Workflow;
    onChange: (key: string) => void;
}

export class WorkflowDrawer extends React.Component<WorkflowDrawerProps, {}> {
	render () {
		let wf = this.props.workflow;
		return (
			<div className='workflow-drawer'>
				<div className='workflow-drawer__section workflow-drawer__labels'> 
					<div className='workflow-drawer__title'>LABELS</div> 
                    <div className='workflow-drawer__labels--list'> 
    		            <WorkflowLabels
    		                workflow={wf}
    		                onChange={key => {
    		                    this.props.onChange(key);
    		                }}
    		            />
                    </div> 
		        </div> 
	            { !wf.status.message ? <span/> : 
                    <div className='workflow-drawer__section workflow-drawer__message'> 
	            		<div className='workflow-drawer__title workflow-drawer__message--label'>
                            MESSAGE
                        </div> 
	            	 	<div className='workflow-drawer__message--content'> 
                             {wf.status.message} 
                        </div> 
	            	</div> 
	            }
		        { !wf.status.resourcesDuration ? <span/> : 
                    <div className='workflow-drawer__section'>
		            	<div className='workflow-drawer__resourcesDuration'> 
			            	<div className='workflow-drawer__title'>
                                RESOURCES DURATION&nbsp;
				            	<a href="https://github.com/argoproj/argo/blob/master/docs/resource-duration.md" target="_blank"> 
									<i className='fas fa-info-circle'/>
								</a> 
                            </div> 
			            	<div className='workflow-drawer__resourcesDuration--container'> 
								<div> 
									<span className='workflow-drawer__resourcesDuration--value'>
                                        {wf.status.resourcesDuration.cpu} sec
                                    </span>
									<span className='workflow-drawer__resourcesDuration--label'>
                                        (*1 CPU)
                                    </span> 
								</div> 
								<div> 
									<span className='workflow-drawer__resourcesDuration--value'>
                                        {wf.status.resourcesDuration.memory} sec
                                    </span>
									<span className='workflow-drawer__resourcesDuration--label'> 
                                        (*100Mi Memory)
                                    </span>
								</div> 
							</div> 
						</div>
                    </div> 
				}
	            { !wf.status.conditions ? <span/> :
                    <div className='workflow-drawer__section'>
		            	<div className='workflow-drawer__title'>
                            CONDITIONS
                        </div> 
		            	<div className='workflow-drawer__conditions'>
			            	{ wf.status.conditions.map((condition) => {
				            	return (
				            		<div className='tag'>
				            			<div className='key'>{condition.type}</div>
				            			<div className='value'>{condition.status}</div> 
				            		</div>
				            	)
				            }) } 
			           	</div> 
			        </div> 
		        }
	        </div>
		)
	}
}