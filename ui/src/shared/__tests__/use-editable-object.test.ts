import {reducer} from '../use-editable-object';

/* Local copy of the literal‑union type for convenience */
type Lang = 'json' | 'yaml';

/* Minimal replica of State<T> for compile‑time checks */
interface TestState<T> {
  object: T;
  serialization: string;
  initialSerialization: string;
  lang: Lang;
  edited: boolean;
}

describe('useEditableObject helper functions', () => {
  const initialSerialization = '{\n  "a": 1,\n  "b": 2\n}';

  const initialState: TestState<Record<string, any>> = {
    object: {a: 1, b: 2},
    serialization: initialSerialization,
    initialSerialization,
    lang: 'json',
    edited: false,
  };

  it('reducer handles setObject action', () => {
    const newObj = {a: 1, b: 3};

    const action = {
      type: 'setObject',
      payload: {
        object: newObj,
        serialization: '{\n  "a": 1,\n  "b": 3\n}',
      },
    } as const;

    const newState = reducer(initialState, action);

    expect(newState.object).toEqual(newObj);
    expect(newState.serialization).toContain('"b": 3');
    expect(newState.edited).toBe(true);
  });

  it('reducer handles setLang action', () => {
    const action = {
      type: 'setLang',
      payload: 'yaml' as Lang,
    } as const;

    const newState = reducer(initialState, action);

    expect(newState.lang).toBe('yaml');
    expect(newState.serialization).toBe('a: 1\nb: 2\n');
    expect(newState.edited).toBe(false);
  });

  it('reducer handles resetObject action', () => {
    const resetObj = {c: 4};

    const action = {
      type: 'resetObject',
      payload: resetObj,
    } as const;

    const newState = reducer({...initialState, edited: true}, action);

    expect(newState.object).toEqual(resetObj);
    expect(newState.serialization).toContain('"c": 4');
    expect(newState.edited).toBe(false);
  });
});