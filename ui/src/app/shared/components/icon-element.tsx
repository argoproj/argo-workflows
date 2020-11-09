import * as React from 'react';
import {Icon} from './icon';

export const IconElement = (props: {icon: Icon}) => <i className={'fa fa-' + props.icon} />;
