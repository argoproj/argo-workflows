import * as React from 'react';
import {uiUrl} from '../../../shared/base';

export const WidgetGallery = ({namespace, name}: {namespace: string; name: string}) => (
    <div className='white-box'>
        <h6>Status Badge</h6>
        <div>
            <iframe frameBorder={0} width={200} height={20} src={uiUrl(`widgets/workflow-status-badges/${namespace}/${name}`)} />
        </div>
        <p>
            <a href={uiUrl(`widgets/workflow-status-badges/${namespace}/${name}?target=_top`)}>
                Open <i className='fa fa-caret-right' />{' '}
            </a>
        </p>
        <h6>Graph</h6>
        <div>
            <iframe frameBorder={0} width={400} height={200} src={uiUrl(`widgets/workflow-graphs/${namespace}/${name}`)} />
        </div>
        <p>
            <a href={uiUrl(`widgets/workflow-graphs/${namespace}/${name}?target=_top&showOptions=false&nodeSize=16`)}>
                Open <i className='fa fa-caret-right' />
            </a>
        </p>
    </div>
);
