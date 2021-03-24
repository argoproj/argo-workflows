import {ObjectMeta} from 'argo-ui/src/models/kubernetes';
import {useEffect, useState} from 'react';
import React = require('react');
import {Link} from '../../../models';
import {services} from '../services';
import {Button} from './button';

export const Links = ({scope, object, button}: {scope: string; object: {metadata: ObjectMeta; status?: any}; button?: boolean}) => {
    const [links, setLinks] = useState<Link[]>();
    const [error, setError] = useState<Error>();

    useEffect(() => {
        services.info
            .getInfo()
            .then(x => (x.links || []).filter(y => y.scope === scope))
            .then(setLinks)
            .catch(setError);
    }, []);

    const formatUrl = (url: string) => {
        const status = object.status || {};
        return url
            .replace(/\${metadata\.namespace}/g, object.metadata.namespace)
            .replace(/\${metadata\.name}/g, object.metadata.name)
            .replace(/\${status\.startedAt}/g, status.startedAt)
            .replace(/\${status\.finishedAt}/g, status.finishedAt);
    };

    const openLink = (url: string) => {
        if ((window.event as MouseEvent).ctrlKey) {
            window.open(url, '_blank');
        } else {
            document.location.href = url;
        }
    };

    return (
        <>
            {error && error.message}
            {links &&
                links.map(({url, name}) => {
                    if (button) {
                        return (
                            <Button onClick={() => openLink(formatUrl(url))} key={name} icon='external-link-alt'>
                                {name}
                            </Button>
                        );
                    }
                    return (
                        <a key={name} href={formatUrl(url)}>
                            {name} <i className='fa fa-external-link-alt' />
                        </a>
                    );
                })}
        </>
    );
};
