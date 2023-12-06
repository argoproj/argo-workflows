import {Page} from 'argo-ui';
import * as React from 'react';
import {uiUrl} from '../../shared/base';
import {Loading} from '../../shared/components/loading';
import {useCollectEvent} from '../../shared/components/use-collect-event';

export const ApiDocs = () => {
    useCollectEvent('openedApiDocs');
    return (
        <Page title='Swagger'>
            <div className='argo-container'>
                <SuspenseSwaggerUI url={uiUrl('assets/openapi-spec/swagger.json')} defaultModelExpandDepth={0} />
            </div>
        </Page>
    );
};

// lazy load SwaggerUI as it is infrequently used and imports very large components (which can be split into a separate bundle)
const LazySwaggerUI = React.lazy(() => {
    import(/* webpackChunkName: "swagger-ui-react-css" */ 'swagger-ui-react/swagger-ui.css');
    return import(/* webpackChunkName: "swagger-ui-react" */ 'swagger-ui-react');
});

function SuspenseSwaggerUI(props: any) {
    return (
        <React.Suspense fallback={<Loading />}>
            <LazySwaggerUI {...props} />
        </React.Suspense>
    );
}
