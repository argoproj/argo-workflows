import React = require('react');

export const ResourcesUsage = (props: {[resource: string]: string}) => (
    <>
        {Object.entries(props || {})
            .map(([resource, quantity]) => quantity + ' ' + resource)
            .join(',')}{' '}
        <a href='https://argoproj.github.io/argo/resources-usage/'>
            <i className='fa fa-info-circle' />
        </a>
    </>
);
