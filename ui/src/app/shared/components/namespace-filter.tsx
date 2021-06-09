import * as React from 'react';
import {useEffect, useState} from 'react';
import {services} from '../services';
import {useLocalStorage} from '../use-local-storage';
import {Utils} from '../utils';
import {WarningIcon} from './fa-icons';
import {InputFilter} from './input-filter';

export const NamespaceFilter = (props: {value: string; onChange: (namespace: string) => void}) => {
    if (Utils.managedNamespace) {
        return <>{Utils.managedNamespace}</>;
    }

    const [namespaces, setNamespaces] = useLocalStorage<string[]>('namespaces', undefined, 60);
    const [error, setError] = useState<Error>();

    useEffect(() => {
        if (!namespaces) {
            services.info
                .listNamespaces()
                .then(r => setNamespaces(r.namespaces))
                .catch(e => setError(e));
        }
    }, []); // no dependencies -> only run once

    // make sure the namespace is allowed
    useEffect(() => {
        if (namespaces && !namespaces.includes(props.value)) {
            props.onChange(namespaces[0]);
        }
    }, [namespaces]);

    if (namespaces) {
        return (
            <select className='argo-field' onChange={e => props.onChange(e.target.value)} value={props.value}>
                {namespaces.map(x => (
                    <option key={x}>{x}</option>
                ))}
            </select>
        );
    } else {
        return (
            <>
                <InputFilter value={props.value} name='ns' onChange={ns => props.onChange(ns)} />
                {error && (
                    <span title={'failed to list namespaces (which in probably just fine): ' + error.toString()}>
                        {' '}
                        <WarningIcon />
                    </span>
                )}
            </>
        );
    }
};
