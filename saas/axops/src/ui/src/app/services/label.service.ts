import * as _ from 'lodash';
import { Injectable } from '@angular/core';
import { Http, URLSearchParams, Headers } from '@angular/http';
import { Observable } from 'rxjs/Observable';

import { Label } from '../model';

@Injectable()
export class LabelService {
    constructor(private http: Http) {
    }

    getLabels(params: {
        type?: string,
        search?: string
        reserved?: boolean
    }, showLoader?: boolean): Observable<Label[]> {
        let customHeader = new Headers();
        if (showLoader) {
            customHeader.append('isUpdated', 'true');
        }
        let search = new URLSearchParams();
        if (params.type) {
            search.set('type', params.type.toString());
        }
        if (typeof params.reserved !== 'undefined') {
            search.set('reserved', params.reserved.toString());
        }
        if (params.search) {
            search.set('search', '~' + params.search.toString());
        }

        return this.http.get(`v1/labels?${search.toString()}`, { headers: customHeader }).map((res) => {
            let result = res.json(), data: Label[] = [];
            // cast objects
            _.forEach(result['data'], (l) => {
                data.push(new Label(l));
            });
            result.data = data;
            return result;
        });
    }
    // Group an array of labels by the key
    groupLabelsByKey(data: Label[]): { labelKey: string, labels: Label[] }[] {
        let tempMap = {}, labelGroups: { labelKey: string, labels: Label[] }[] = [];

        // Group all label objects by label->key
        _.forEach(data, (label) => {
            if (typeof tempMap[label.key] === 'undefined') {
                tempMap[label.key] = [];
            }
            tempMap[label.key].push(label);
        });

        // mutate the map into an array as angular works better with arrays
        _.forEach(tempMap, (value, key) => {
            labelGroups.push({ labelKey: <string>key, labels: <Label[]>value });
        });
        return labelGroups;
    }

    createLabel(labelName: string, type: string) {
        let label = new Label();
        label.key = labelName;
        label.type = type;
        return this.http.post(`v1/labels`, label).map(res => res.json());
    }

    removeLabel(id: string) {
        let search = new URLSearchParams();
        search.set('id', id);

        return this.http.delete(`v1/labels?${search.toString()}`).map(res => res.json());
    }
}
