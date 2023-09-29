import {AppContext} from 'argo-ui/src/index';
import * as PropTypes from 'prop-types';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';

/**
 * @deprecated Use React hooks instead.
 */
export class BasePage<P extends RouteComponentProps<any>, S> extends React.Component<P, S> {
    public static contextTypes = {
        router: PropTypes.object,
        apis: PropTypes.object
    };

    public queryParam(name: string) {
        return this.params.get(name);
    }

    private get params() {
        return new URLSearchParams(this.appContext.router.route.location.search);
    }

    public queryParams(name: string) {
        return this.params.getAll(name);
    }

    public set url(url: string) {
        this.appContext.router.history.push(url);
    }

    protected get appContext() {
        return this.context as AppContext;
    }
}
