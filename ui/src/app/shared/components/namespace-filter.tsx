import * as React from 'react';
import {Utils} from '../utils';
import {InputFilter} from './input-filter';

export const NamespaceFilter = (props: {value: string; onChange: (namespace: string) => void}) =>
    Utils.managedNamespace ? <>{Utils.managedNamespace}</> : <InputFilter value={props.value} name='ns' onChange={ns => props.onChange(ns)} />;
