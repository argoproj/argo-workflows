import * as React from 'react';

import * as nsUtils from '../namespaces';
import {InputFilter} from './input-filter';

export const NamespaceFilter = (props: {value: string; onChange: (namespace: string) => void}) =>
    nsUtils.getManagedNamespace() ? <>{nsUtils.getManagedNamespace()}</> : <InputFilter value={props.value} name='ns' onChange={ns => props.onChange(ns)} />;
