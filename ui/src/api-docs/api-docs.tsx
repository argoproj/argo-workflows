import {Page} from 'argo-ui/src/components/page/page';
import * as React from 'react';

import {uiUrl} from '../shared/base';
import {ZeroState} from '../shared/components/zero-state';
import {useCollectEvent} from '../shared/use-collect-event';

export function ApiDocs() {
    useCollectEvent('openedApiDocs');
    return (
        <Page
            title='Swagger'
            toolbar={{
                breadcrumbs: [{title: 'Swagger', path: uiUrl('apidocs')}]
            }}>
            <ZeroState title='Swagger'>
                <p>
                    Download the <a download={uiUrl('assets/openapi-spec/swagger.json')}>Open API / Swagger spec</a>.
                </p>
                <p>
                    Download the <a download={uiUrl('assets/jsonschema/schema.json')}>JSON schema</a>.
                </p>
                <p>
                    View the interactive Swagger UI{' '}
                    <a href='https://argo-workflows.readthedocs.io/en/latest/swagger/' target='_blank' rel='noreferrer'>
                        in the Documentation
                    </a>
                    .
                </p>
            </ZeroState>
        </Page>
    );
}
