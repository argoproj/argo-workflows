import * as classNames from 'classnames';
import * as React from 'react';
import {NodePhase} from '../../../models';
import {Utils} from '../utils';

export const PhaseIcon = ({value}: {value: NodePhase}) => {
    return <i className={classNames('fa', Utils.statusIconClasses(value))} aria-hidden='true' />;
};
