import {useState} from 'react';

/**
 * useEditableObject is a React hook to manage the state of object that can be edited and updated.
 * Uses ref comparisons to determine whether the resource has been edited.
 */
export function useEditableObject<T>(initial?: T): [T, boolean, React.Dispatch<T>, (value: T) => void] {
    const [value, setValue] = useState<T>(initial);
    const [initialValue, setInitialValue] = useState<T>(initial);

    // Note: This is a pure reference comparison instead of a deep comparison for performance
    // reasons, since <ObjectEditor> changes are latency-sensitive.
    const edited = value !== initialValue;

    function resetValue(value: T) {
        setValue(value);
        setInitialValue(value);
    }

    return [value, edited, setValue, resetValue];
}
