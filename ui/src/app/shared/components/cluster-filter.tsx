import * as React from 'react';
import {services} from '../services';
import {DataLoaderDropdown} from './data-loader-dropdown';

export const ClusterFilter = ({value, onChange}: {value: string; onChange: (cluster: string) => void}) => (
    <DataLoaderDropdown value={value} load={() => services.info.getInfo().then(list => list.clusters)} onChange={onChange} />
);
