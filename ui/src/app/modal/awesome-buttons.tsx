import * as React from 'react';
import {Icon} from '../shared/components/icon';
import {AwesomeButton} from './awesome-button';

export const AwesomeButtons = ({options, dismiss}: {options: {[key: string]: string}; dismiss: () => void}) => (
    <>
        {Object.entries(options).map(([icon, title]) => (
            <AwesomeButton icon={icon as Icon} title={title} key={icon} />
        ))}
        <p>
            <a onClick={() => dismiss()}>Maybe later...</a>
        </p>
    </>
);
