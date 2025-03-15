import {useReducer} from 'react';

import {Lang, parse, stringify} from '../shared/components/object-parser';
import {ScopedLocalStorage} from '../shared/scoped-local-storage';

type SetObjectAction<T> = {type: 'setObject'; payload: {object: T; serialization?: string}};
type Action<T> = {type: 'setLang'; payload: Lang} | SetObjectAction<T> | {type: 'resetObject'; payload: T};

interface State<T> {
    /** The parsed form of the object, kept in sync with "serialization" */
    object: T;
    /** The stringified form of the object, kept in sync with "object" */
    serialization: string;
    /** The serialization language used (YAML or JSON) */
    lang: Lang;
    /** The initial stringified form of the object. Used to check if it was edited */
    initialSerialization: string;
    /** Whether any changes have been made */
    edited: boolean;
}

const defaultLang = 'yaml';
const storage = new ScopedLocalStorage('object-editor');

export function reducer<T>(state: State<T>, action: Action<T>) {
    const newState = {...state};
    switch (action.type) {
        case 'setObject':
            newState.object = action.payload.object;
            newState.serialization = action.payload.serialization ?? stringify(newState.object, newState.lang);
            newState.edited = newState.initialSerialization !== newState.serialization;
            return newState;
        case 'resetObject':
            newState.object = action.payload;
            newState.serialization = stringify(newState.object, newState.lang);
            newState.initialSerialization = newState.serialization;
            newState.edited = false;
            return newState;
        case 'setLang':
            newState.lang = action.payload;
            storage.setItem('lang', newState.lang, defaultLang);
            newState.serialization = stringify(newState.object, newState.lang);
            if (!newState.edited) {
                newState.initialSerialization = newState.serialization;
            }
            return newState;
    }
}

export function createInitialState<T>(object?: T): State<T> {
    const lang = storage.getItem('lang', defaultLang);
    const serialization = object ? stringify(object, lang) : null;
    return {
        object,
        serialization,
        lang,
        initialSerialization: serialization,
        edited: false
    };
}

/**
 * Action creator for setObject that can accept a string and parse it.
 * The reason the parsing logic isn't in the reducer is because we want parse
 * errors to be propagated to the caller.
 */
export function setObjectActionCreator<T>(value: string | T): SetObjectAction<T> {
    return {
        type: 'setObject',
        payload: typeof value === 'string' ? {object: parse<T>(value), serialization: value} : {object: value}
    };
}

/**
 * useEditableObject is a React hook to manage the state of an object that can be serialized and edited, encapsulating the logic to
 * parse/stringify the object as necessary.
 */
export function useEditableObject<T>(object?: T): State<T> & {
    setObject: (value: string | T) => void;
    resetObject: (value: T) => void;
    setLang: (lang: Lang) => void;
} {
    const [state, dispatch] = useReducer(reducer<T>, object, createInitialState);
    return {
        ...state,
        setObject: value => dispatch(setObjectActionCreator<T>(value)),
        resetObject: value => dispatch({type: 'resetObject', payload: value}),
        setLang: value => dispatch({type: 'setLang', payload: value})
    };
}
