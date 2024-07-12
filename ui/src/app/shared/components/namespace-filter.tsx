import * as React from 'react';
import {nsUtils} from '../namespaces';
import {InputFilter} from './input-filter';

export const NamespaceFilter = (props: {value: string; onChange: (namespace: string) => void}) =>
    nsUtils.managedNamespace ? <>{nsUtils.managedNamespace}</> : <InputFilter value={props.value} name='ns' onChange={ns => props.onChange(ns)} />;
