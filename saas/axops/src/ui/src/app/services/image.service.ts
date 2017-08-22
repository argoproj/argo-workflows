import {Inject, Injectable} from '@angular/core';
import {Http, Headers} from '@angular/http';

@Injectable()
export class ImageService {
    constructor(@Inject(Http) private _http) {
    }

    getImagesAsync() {
        return this._http.get('v1/images')
            .map( res => res.json());
    }

    deleteImageByIdAsync(ax_id: string) {
        return this._http.delete(`v1/images/${ax_id}`);
    }

    importImageAsync(form) {
        let customHeader = new Headers();
        customHeader.append('isUpdated', 'true');
        return this._http.post(`v1/images`, JSON.stringify(form.value), { headers: customHeader });
    }

    searchAsync(searchString = '') {
        return this._http.get(`v1/images?search=~${searchString}&session`)
            .map(res => res.json());
    }
}
