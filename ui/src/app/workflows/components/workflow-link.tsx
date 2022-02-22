import * as React from 'react';
import {uiUrl} from '../../shared/base';
import {LinkButton} from '../../shared/components/link-button';

export const WorkflowLink = (props: {namespace: string; name: string}) => (
    <LinkButton to={uiUrl('workflows/' + props.namespace + '/' + props.name)}>
        <i className='fa fa-stream' /> {props.name}
    </LinkButton>
);
