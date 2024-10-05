import {useState} from 'react';

export function useCustomLocalStorage<T>(key: string, initial: T, onError: (err: any) => T | undefined): [T | undefined, React.Dispatch<T>] {
    const [storedValue, setStoredValue]: [T | undefined, React.Dispatch<T>] = useState(() => {
        if (window === undefined) {
            return initial;
        }
        try {
            const item = window.localStorage.getItem(key);
            // try retrieve if none present, default to initial
            return item ? JSON.parse(item) : initial;
        } catch (err) {
            const val = onError(err) || undefined;
            if (val === undefined) {
                return undefined;
            }
            return val;
        }
    });

    const setValue = (value: T | ((oldVal: T) => T)) => {
        try {
            const valueToStore = value instanceof Function ? value(storedValue) : value;
            if (window !== undefined) {
                window.localStorage.setItem(key, JSON.stringify(valueToStore));
                setStoredValue(valueToStore);
            }
        } catch (err) {
            const val = onError(err) || undefined;
            if (val === undefined) {
                return undefined;
            }
            return val;
        }
    };

    return [storedValue, setValue];
}

export function useLocalStorage<T>(key: string, initial: T): [T, React.Dispatch<T>] {
    return useCustomLocalStorage(key, initial, () => initial);
}
