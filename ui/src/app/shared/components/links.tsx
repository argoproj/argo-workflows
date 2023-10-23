import {ObjectMeta} from 'argo-ui/src/models/kubernetes';
import {useEffect, useState} from 'react';
import * as React from 'react';
import {Link, Workflow} from '../../../models';
import {services} from '../services';
import {Button} from './button';

function toEpoch(datetime: string) {
    if (datetime) {
        return new Date(datetime).getTime();
    } else {
        return Date.now();
    }
}

function addEpochTimestamp(jsonObject: {metadata: ObjectMeta; workflow?: Workflow; status?: any}) {
    if (jsonObject === undefined || jsonObject.status.startedAt === undefined) {
        return;
    }

    jsonObject.status.startedAtEpoch = toEpoch(jsonObject.status.startedAt);
    jsonObject.status.finishedAtEpoch = toEpoch(jsonObject.status.finishedAt);
}

export function processURL(urlExpression: string, jsonObject: any) {
    addEpochTimestamp(jsonObject);
    /* replace ${} from input url with corresponding elements from object
    only return null for known variables, otherwise empty string*/
    return urlExpression.replace(/\${[^}]*}/g, x => {
        const parts = x.replace(/(\$%7B|%7D|\${|})/g, '').split('.');
        const emptyVal = parts[0] === 'workflow' ? '' : null;
        const res = parts.reduce((p: any, c: string) => (p && p[c]) || emptyVal, jsonObject);
        return res;
    });
}

export function openLinkWithKey(url: string) {
    if ((window.event as MouseEvent).ctrlKey || (window.event as MouseEvent).metaKey) {
        window.open(url, '_blank');
    } else {
        document.location.href = url;
    }
}

export function Links({scope, object, button}: {scope: string; object: {metadata: ObjectMeta; workflow?: Workflow; status?: any}; button?: boolean}) {
    const [links, setLinks] = useState<Link[]>();
    const [error, setError] = useState<Error>();

    useEffect(() => {
        services.info
            .getInfo()
            .then(x => (x.links || []).filter(y => y.scope === scope))
            .then(setLinks)
            .catch(setError);
    }, []);

    return (
        <>
            {error && error.message}
            {links &&
                links.map(({url, name}) => {
                    if (button) {
                        return (
                            <Button onClick={() => openLinkWithKey(processURL(url, object))} key={name} icon='external-link-alt'>
                                {name}
                            </Button>
                        );
                    }
                    return (
                        <a key={name} href={processURL(url, object)}>
                            {name} <i className='fa fa-external-link-alt' />
                        </a>
                    );
                })}
        </>
    );
}
