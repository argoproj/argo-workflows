import {useEffect} from 'react';
import {services} from '../../shared/services';

export function useCollectEvent(name: string) {
    useEffect(() => {
        services.info.collectEvent(name);
    }, []);
}
