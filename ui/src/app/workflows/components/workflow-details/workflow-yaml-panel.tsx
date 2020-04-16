import * as jsYaml from 'js-yaml';
import * as React from 'react';
import {Workflow} from '../../../../models';
import {YamlViewer} from '../../../shared/components/yaml/yaml-viewer';

export const WorkflowYamlPanel = (props: {workflow: Workflow}) => (
    <div className='white-box'>
        <div className='white-box__details'>
            <YamlViewer value={jsYaml.dump(props.workflow)} />
        </div>
    </div>
);
