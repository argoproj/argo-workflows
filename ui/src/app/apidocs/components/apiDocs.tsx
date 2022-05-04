import {Page} from 'argo-ui';
import * as React from 'react';
import SwaggerUI from 'swagger-ui-react';
import 'swagger-ui-react/swagger-ui.css';
import {uiUrl} from '../../shared/base';
import {useCollectEvent} from '../../shared/components/use-collect-event';

export const ApiDocs = () => {
    useCollectEvent('openedApiDocs');
    return (
        <Page title='Swagger'>
            <div className='argo-container'>
                <SwaggerUI url={uiUrl('assets/openapi-spec/swagger.json')} defaultModelExpandDepth={0} />
            </div>
        </Page>
    );
};
