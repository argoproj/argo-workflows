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

    // this allows us to set-multiple parameters at once
    public setQueryParams(newParams: any) {
        const params = this.params;
        Object.keys(newParams).forEach(name => {
            const value = newParams[name];
            if (value !== null) {
                params.set(name, value);
            } else {
                params.delete(name);
            }
        });
        this.pushParams(params);
    }

    public clearQueryParams() {
        this.url = this.props.match.url;
    }

    // this allows us to set-multiple parameters at once
    public appendQueryParams(newParams: {name: string; value: string}[]) {
        const params = this.params;
        newParams.forEach(param => params.delete(param.name));
        newParams.forEach(param => params.append(param.name, param.value));
        this.pushParams(params);
    }

    private pushParams(params: URLSearchParams) {
        this.url = `${this.props.match.url}?${params.toString()}`;
    }

    public set url(url: string) {
        this.appContext.router.history.push(url);
    }

    protected get appContext() {
        return this.context as AppContext;
    }
}
