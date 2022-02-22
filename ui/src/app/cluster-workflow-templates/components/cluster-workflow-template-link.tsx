import * as React from 'react';
import {uiUrl} from '../../shared/base';
import {LinkButton} from '../../shared/components/link-button';

export const ClusterWorkflowTemplateLink = (props: {name: string}) => (
    <LinkButton to={uiUrl('cluster-workflow-templates/' + props.name)}>
        <i className='fa fa-window-restore' /> {props.name}
    </LinkButton>
);
