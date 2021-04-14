import {ObjectMeta} from 'argo-ui/src/models/kubernetes';
import {Liquid} from 'liquidjs';
import {useEffect, useState} from 'react';
import React = require('react');
import {Link, Workflow} from '../../../models';
import {services} from '../services';
import {Button} from './button';

const engine = new Liquid({
    outputDelimiterLeft: '${',
    outputDelimiterRight: '}'
});

export const ProcessURL = (url: string, jsonObject: any) => {
    /* replace ${} from input url with corresponding elements from object
    return null if element is not found*/
    return engine.parseAndRenderSync(url, jsonObject);
};

export const Links = ({scope, object, button}: {scope: string; object: {metadata: ObjectMeta; workflow?: Workflow; status?: any}; button?: boolean}) => {
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
        return ProcessURL(url, object);
    };

    const openLink = (url: string) => {
        if ((window.event as MouseEvent).ctrlKey || (window.event as MouseEvent).metaKey) {
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
