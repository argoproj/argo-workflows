import { Headers } from '@angular/http';
import { HttpHeaders } from '@angular/common/http';

export interface AxRequestOptions {
    noLoader?: boolean;
    noErrorHandling?: boolean;
}

export class AxHeaders extends Headers implements AxRequestOptions {

    constructor(headers?: Headers | AxRequestOptions) {
        if (headers instanceof Headers) {
            super(headers);
        } else {
            super();
        }
        Object.assign(this, headers);
    }

    public get noLoader(): boolean {
        return this.has('isUpdated');
    }

    public get noErrorHandling(): boolean {
        return this.has('noErrorHandling');
    }

    public set noLoader(value: boolean) {
        this.setFlagHeader('isUpdated', value);
    }

    public set noErrorHandling(value: boolean) {
        this.setFlagHeader('noErrorHandling', value);
    }

    private setFlagHeader(name: string, value: boolean): AxHeaders {
        if (value) {
            this.set(name, 'true');
        } else {
            this.delete(name);
        }
        return this;
    }
}
