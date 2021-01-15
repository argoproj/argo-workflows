import {ObjectMeta} from 'argo-ui/src/models/kubernetes';
import {useEffect, useState} from 'react';
import React = require('react');
import {Link} from '../../../models';
import {services} from '../services';

export const Links = ({scope, object}: {scope: string; object: {metadata: ObjectMeta; status?: any}}) => {
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

    return (
        <>
            {error && error.message}
            {links &&
                links.map(({url, name}) => (
                    <a key={name} href={formatUrl(url)}>
                        {name} <i className='fa fa-external-link-alt' />
                    </a>
                ))}
        </>
    );
};
