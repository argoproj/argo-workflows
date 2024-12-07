import {createInitialState, reducer, setObjectActionCreator} from './use-editable-object';

describe('createInitialState', () => {
    test('without object', () => {
        expect(createInitialState()).toEqual({
            object: undefined,
            serialization: null,
            lang: 'yaml',
            initialSerialization: null,
            edited: false
        });
    });

    test('with object', () => {
        expect(createInitialState({a: 1})).toEqual({
            object: {a: 1},
            serialization: 'a: 1\n',
            lang: 'yaml',
            initialSerialization: 'a: 1\n',
            edited: false
        });
    });
});

describe('reducer', () => {
    const testState = {
        object: {a: 1},
        serialization: 'a: 1\n',
        lang: 'yaml',
        initialSerialization: 'a: 1\n',
        edited: false
    } as const;

    test('setLang unedited', () => {
        const newState = reducer(testState, {type: 'setLang', payload: 'json'});
        expect(newState).toEqual({
            object: {a: 1},
            serialization: '{\n  "a": 1\n}',
            lang: 'json',
            initialSerialization: '{\n  "a": 1\n}',
            edited: false
        });
    });

    test('setLang edited', () => {
        const newState = reducer(
            {
                ...testState,
                edited: true
            },
            {type: 'setLang', payload: 'json'}
        );
        expect(newState).toEqual({
            object: {a: 1},
            serialization: '{\n  "a": 1\n}',
            lang: 'json',
            initialSerialization: 'a: 1\n',
            edited: true
        });
    });

    test('setObject with object and string', () => {
        const newState = reducer(testState, {type: 'setObject', payload: {object: {a: 2}, serialization: 'a: 2'}});
        expect(newState).toEqual({
            object: {a: 2},
            serialization: 'a: 2',
            lang: 'yaml',
            initialSerialization: 'a: 1\n',
            edited: true
        });
    });

    test('setObject with only object', () => {
        const newState = reducer(testState, {type: 'setObject', payload: {object: {a: 2}}});
        expect(newState).toEqual({
            object: {a: 2},
            serialization: 'a: 2\n',
            lang: 'yaml',
            initialSerialization: 'a: 1\n',
            edited: true
        });
    });

    test('resetObject with object', () => {
        const newState = reducer(testState, {type: 'resetObject', payload: {a: 2}});
        expect(newState).toEqual({
            object: {a: 2},
            serialization: 'a: 2\n',
            lang: 'yaml',
            initialSerialization: 'a: 2\n',
            edited: false
        });
    });
});

describe('setObjectActionCreator', () => {
    test('with object', () => {
        expect(setObjectActionCreator({a: 2})).toEqual({
            type: 'setObject',
            payload: {object: {a: 2}}
        });
    });

    test('with string', () => {
        expect(setObjectActionCreator('a: 2')).toEqual({
            type: 'setObject',
            payload: {object: {a: 2}, serialization: 'a: 2'}
        });
    });

    test("with string that can't be parsed", () => {
        expect(() => setObjectActionCreator('@')).toThrow('Plain value cannot start with reserved character @');
    });
});
