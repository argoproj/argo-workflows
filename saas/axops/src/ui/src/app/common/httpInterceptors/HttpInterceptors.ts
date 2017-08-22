import { Injectable, Injector } from '@angular/core';
import { Http, ConnectionBackend, RequestOptions, Request, RequestOptionsArgs, Response, Headers } from '@angular/http';
import { Observable } from 'rxjs/Rx';
import { Router } from '@angular/router';
import { ToasterService } from 'angular2-toaster/angular2-toaster';

import { LoaderService, AxHeaders, CookieService, AuthenticationService } from '../../services';

export const ERROR_TYPE = {
    ERR_API_ACCOUNT_NOT_CONFIRMED: 'ERR_API_ACCOUNT_NOT_CONFIRMED',
    ERR_API_RESOURCE_NOT_FOUND: 'ERR_API_RESOURCE_NOT_FOUND'
};

@Injectable()
export class HttpInterceptor extends Http {
    constructor(private backend: ConnectionBackend,
        private defaultOptions: RequestOptions,
        private router: Router,
        private loaderService: LoaderService,
        private toasterService: ToasterService,
        private cookieService: CookieService,
        private injector: Injector) {

        super(backend, defaultOptions);
    }

    public get(url: string, options?: RequestOptionsArgs): Observable<Response> {
        return this.processRequest(newOptions => super.get(url, newOptions), url, options);
    }

    public put(url: string, body: any, options?: RequestOptionsArgs): Observable<Response> {
        return this.processRequest(newOptions => super.put(url, body, options), url, options);
    }

    public post(url: string, body: any, options?: RequestOptionsArgs): Observable<Response> {
        return this.processRequest(newOptions => super.post(url, body, options), url, options);
    }

    public delete(url: string, options?: RequestOptionsArgs): Observable<Response> {
        return this.processRequest(newOptions => super.delete(url, options), url, options);
    }

    private processRequest(
        executor: (options: RequestOptionsArgs) => Observable<Response>,
        url: string | Request,
        options?: RequestOptionsArgs) {
        options = options || {};
        if (!options.headers) {
            options.headers = new Headers();
        }

        if (!options.headers.get('Content-Type')) {
            options.headers.append('Content-Type', 'application/json');
        }
        let headers = new AxHeaders(options.headers);

        if (headers.noLoader) {
            this.loaderService.hide.emit(url);
        } else {
            this.loaderService.show.emit(url);
        }

        return executor(options)
            .map(res => {
                this.loaderService.hide.emit(url);
                return res;
            }).catch((err) => {
                this.loaderService.hide.emit(url);
                if (!headers.noErrorHandling) {
                    this.handleHTTPError(err);
                }
                return Observable.throw(err);
            });
    }

    /**
     * Specific handler for taking care of 401 error code if needed
     */
    private process401Error(err) {
        // For case when user has not confirmed his email address, we redirect him to a different page
        if (err.code === ERROR_TYPE.ERR_API_ACCOUNT_NOT_CONFIRMED) {
            this.router.navigateByUrl('/setup/confirm');
            this.displayError(err);
        } else {
            this.displayError(err);
            this.injector.get(AuthenticationService).redirectUnauthenticatedUser();
        }
    }
    /**
     * Send the error object this way and it will pop the toaster for you
     */
    private displayError(error) {
        this.toasterService.pop('error', error.code ? error.code : 'Internal error encountered.', error.message ? error.message : '');
    }

    /**
     * Common place to process all XHR errors
     */
    private handleHTTPError(err) {
        let error = err.json();
        if (err.status === 401) {
            this.process401Error(error);
        } else {
            this.displayError(error);
        }
    }
}
