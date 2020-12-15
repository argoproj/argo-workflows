import React = require('react');
import {useEffect, useState} from 'react';
import {Link} from '../../../models';
import {services} from '../services';

export const ChatButton = () => {
    const [link, setLink] = useState<Link>();

    useEffect(() => {
        services.info
            .getInfo()
            .then(info => info.links)
            .then(links => links.filter(x => x.scope === 'chat'))
            .then(links => {
                if (links.length > 0) {
                    setLink(links[0]);
                }
            });
    }, []);

    if (!link) {
        return null;
    }

    return (
        <div style={{position: 'fixed', right: 10, bottom: 10}}>
            <a href={link.url} className='argo-button argo-button--special'>
                <i className='fas fa-comment-alt' /> {link.name}
            </a>
        </div>
    );
};
