import * as React from 'react';
import {MemoizationStatus} from '../../../../models';
require('./memoization-info.scss');

export const MemoizationInfo = ({memStat}: {memStat: MemoizationStatus}) => (
    <div className='mi'>
        <div className='mi--row'>
            <div className='mi--row__label'>CACHE NAME</div>
            <div className='mi--row__data'>{memStat.cacheName}</div>
        </div>
        <div className='mi--row'>
            <div className='mi--row__label'>KEY</div>
            <div className='mi--row__data'>{memStat.key}</div>
        </div>
        <div className='mi--row'>
            <div className='mi--row__label'>HIT?</div>
            <div className='mi--row__data'>{memStat.hit ? 'yes' : 'no'}</div>
        </div>
    </div>
);
