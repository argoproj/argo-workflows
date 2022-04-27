import {Page} from 'argo-ui';
import * as React from 'react';
import SwaggerUI from 'swagger-ui-react';
import 'swagger-ui-react/swagger-ui.css';
import {uiUrl} from '../../shared/base';
import {TrackEvent} from '../../shared/components/track-user-interface-event';

export const ApiDocs = () => (
    <Page title='Swagger'>
        <div className='argo-container'>
            <SwaggerUI url={uiUrl('assets/openapi-spec/swagger.json')} defaultModelExpandDepth={0} />
        </div>
        <TrackEvent name={'openedApiDocs'} />
    </Page>
);
