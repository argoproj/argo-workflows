import * as React from 'react';
import {InputFilter} from './input-filter';

export const NamespaceFilter = (props: {value: string; onChange: (namespace: string) => void}) => <InputFilter value={props.value} name='ns' onChange={ns => props.onChange(ns)} />;
