import classNames from 'classnames';

import * as React from 'react';
import {Utils} from '../utils';

export function PhaseIcon({value}: {value: string}) {
    return <i className={classNames('fa', Utils.statusIconClasses(value))} aria-hidden='true' />;
}
