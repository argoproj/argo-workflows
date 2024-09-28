import {useState} from 'react';

/**
 * useEditableResource is a hook to manage the state of a resource that be edited and updated.
 */
export function useEditableResource<T>(initial?: T): [T, boolean, React.Dispatch<T>, (value: T) => void] {
    const [value, setValue] = useState<T>(initial);
    const [initialValue, setInitialValue] = useState<T>(initial);

    // TODO: Fix this so that it handles object comparison properly.
    // Currently, this returns true if you make a change and immediately undo it, or if you save your changes.
    // This could be solved using "const edited = JSON.stringify(value) !== JSON.stringify(initialValue)",
    // but that has a performance penalty.
    const edited = value !== initialValue;

    function resetValue(value: T) {
        setValue(value);
        setInitialValue(value);
    }

    return [value, edited, setValue, resetValue];
}
