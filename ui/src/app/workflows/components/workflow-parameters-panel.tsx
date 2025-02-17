import * as React from 'react';

import {Parameter} from '../../../models';

export const WorkflowParametersPanel = (props: {parameters: Parameter[]}) => (
    <div className='white-box'>
        <div className='white-box__details'>
            {props.parameters.map(param => (
                <div className='row white-box__details-row' key={param.name}>
                    <div className='columns small-3'>{param.name}</div>
                    <div className='columns small-9'>{param.value}</div>
                </div>
            ))}
        </div>
    </div>
);
