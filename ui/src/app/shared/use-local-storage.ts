import {Dispatch, SetStateAction} from 'react';

interface Item<T> {
    value: T;
    storedAt: string; // date gets converted into string by JSON.parse
}

// the signature is meant to be very similar to React.useState()
export const useLocalStorage = <S>(key: string, defaultValue: S = undefined, maxAgeSeconds: number = undefined): [S | undefined, Dispatch<SetStateAction<S | undefined>>] => {
    const set = (v: S) => localStorage.setItem(key, JSON.stringify({value: v, storedAt: new Date()}));
    const text = localStorage.getItem(key);
    if (text) {
        const x = JSON.parse(text) as Item<S>;
        if (!maxAgeSeconds || new Date(x.storedAt).getTime() > new Date().getTime() - maxAgeSeconds * 1000) {
            return [x.value, set];
        }
    }
    return [defaultValue, set];
};
