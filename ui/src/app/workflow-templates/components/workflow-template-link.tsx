import * as React from 'react';
import {uiUrl} from '../../shared/base';
import {LinkButton} from '../../shared/components/link-button';

export const WorkflowTemplateLink = (props: {namespace: string; name: string}) => (
    <LinkButton to={uiUrl('workflow-templates/' + props.namespace + '/' + props.name)}>
        <i className='fa fa-window-maximize' /> {props.name}
    </LinkButton>
);
