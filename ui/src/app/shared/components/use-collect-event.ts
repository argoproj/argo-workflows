import {useEffect} from 'react';
import {services} from '../../shared/services';

export const useCollectEvent = (name: string) => {
    useEffect(() => {
        services.info.collectEvent(name).then();
    }, []);
};
