import * as React from 'react';
import {MemoizationStatus} from '../../../models';

export const MemoizationInfo = ({memStat}: {memStat: MemoizationStatus}) => (
    <div>
        <div>CACHE NAME: {memStat.cacheName}</div>
        <div>KEY: {memStat.key}</div>
        <div>HIT?: {memStat.hit ? 'yes' : 'no'}</div>
    </div>
);
