import * as moment from 'moment';
import { Observable } from 'rxjs';
import { EventEmitter, Injectable, Output } from '@angular/core';
import { Http, URLSearchParams } from '@angular/http';

import { Rule, NotificationEvent } from '../model';
import { AxHeaders } from './headers';

@Injectable()
export class NotificationService {
    @Output() showNotification: EventEmitter<any> = new EventEmitter();

    constructor(private http: Http) {
    }

    public async getChannels(): Promise<string[]> {
        return this.http.get(`v1/notification_center/channels`, {headers: new AxHeaders({ noLoader: true })}).toPromise().then(res => res.json().data);
    }

    public async getRules(): Promise<Rule[]> {
        return this.http.get(`v1/notification_center/rules`, {headers: new AxHeaders({ noLoader: true })}).toPromise().then(res => res.json().data);
    }

    public async createRule(rule: Rule) {
        return this.http.post(`v1/notification_center/rules`, JSON.stringify(rule)).toPromise().then(res => res.json());
    }

    public async updateRule(rule: Rule) {
        return this.http.put(`v1/notification_center/rules/${rule.rule_id}`, JSON.stringify(rule)).toPromise().then(res => res.json());
    }

    public async deleteRule(rule_id: string) {
        return this.http.delete(`v1/notification_center/rules/${rule_id}`).toPromise();
    }

    public async getSeverities(): Promise<string[]> {
        return this.http.get(`v1/notification_center/severities`, {headers: new AxHeaders({ noLoader: true })}).toPromise().then(res => res.json().data);
    }

    public async getEvents(params?: {
        channel?: string,
        facility?: string,
        severity?: string,
        trace_id?: string,
        recipient?: string,
        ordering?: string,
        order_by?: string,
        min_time?: number,
        max_time?: number,
        limit?: number,
        offset?: number,
    }, hideLoader = true): Promise<NotificationEvent[]> {
        let filter = new URLSearchParams();

        if (params.channel) {
            filter.set('channel', params.channel.toString());
        }

        if (params.facility) {
            filter.set('facility', params.facility.toString());
        }

        if (params.severity) {
            filter.set('severity', params.severity.toString());
        }

        if (params.trace_id) {
            filter.set('trace_id', params.trace_id.toString());
        }

        if (params.recipient) {
            filter.set('recipient', params.recipient.toString());
        }

        if (params.ordering) {
            filter.set('ordering', params.ordering.toString());
        }

        if (params.order_by) {
            filter.set('order_by', params.order_by.toString());
        }

        if (params.min_time) {
            filter.set('min_time', params.min_time.toString());
        }

        if (params.max_time) {
            filter.set('max_time', params.max_time.toString());
        }

        if (params.limit) {
            filter.set('limit', params.limit.toString());
        }

        if (params.offset) {
            filter.set('offset', params.offset.toString());
        }

        return this.http.get(`v1/notification_center/events`, {headers: new AxHeaders({ noLoader: hideLoader }), search: filter}).toPromise().then(res => res.json().data);
    }

    public getEventsStream(username: string) {
        let mostRecentCheckTime = moment().unix();
        return Observable.interval(5000).flatMap(() => {
            let result = Observable.fromPromise(this.getEvents({
                recipient: username,
                min_time: mostRecentCheckTime
            }).catch(e => [])).flatMap(event => event);
            mostRecentCheckTime = moment().unix();
            return result;
        });
    }

    public acknowledgeNotification(id: string): Promise<NotificationEvent> {
        return this.http.put(`v1/notification_center/events/${id}/read`, null).toPromise().then(res => <NotificationEvent> res.json());
    }
}
