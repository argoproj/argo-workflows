import {languages} from 'monaco-editor/esm/vs/editor/editor.api';
import * as React from 'react';
import {createRef, useEffect, useState} from 'react';
import MonacoEditor from 'react-monaco-editor';
import {uiUrl} from '../../base';
import {ScopedLocalStorage} from '../../scoped-local-storage';
import {parse, stringify} from '../object-parser';
import {PhaseIcon} from '../phase-icon';
import {ToggleButton} from '../toggle-button';

interface Props<T> {
    type?: string;
    value: T;
    buttons?: React.ReactNode;
    onChange?: (value: T) => void;
}

const defaultLang = 'yaml';

export const ObjectEditor = <T extends any>({type, value, buttons, onChange}: Props<T>) => {
    const storage = new ScopedLocalStorage('object-editor');
    const [error, setError] = useState<Error>();
    const [lang, setLang] = useState<string>(storage.getItem('lang', defaultLang));
    const [text, setText] = useState<string>(stringify(value, lang));

    useEffect(() => storage.setItem('lang', lang, defaultLang), [lang]);
    useEffect(() => setText(stringify(value, lang)), [value]);
    useEffect(() => setText(stringify(parse(text), lang)), [lang]);
    useEffect(() => {
        // we ONLY want to change the text, if the normalized version has changed, this prevents white-space changes
        // from resulting in a significant change
        if (text !== stringify(parse(editor.current.editor.getValue()), lang)) {
            editor.current.editor.setValue(text);
        }
    }, [text, lang]);

    useEffect(() => {
        if (type && lang === 'json') {
            const uri = uiUrl('assets/jsonschema/schema.json');
            fetch(uri)
                .then(res => res.json())
                .then(swagger => {
                    // adds auto-completion to JSON only
                    languages.json.jsonDefaults.setDiagnosticsOptions({
                        validate: true,
                        schemas: [
                            {
                                uri,
                                fileMatch: ['*'],
                                schema: {
                                    $id: 'http://workflows.argoproj.io/' + type + '.json',
                                    $ref: '#/definitions/' + type,
                                    $schema: 'http://json-schema.org/draft-07/schema',
                                    definitions: swagger.definitions
                                }
                            }
                        ]
                    });
                })
                .catch(setError);
        }
    }, [lang, type]);

    const editor = createRef<MonacoEditor>();

    return (
        <>
            <div style={{paddingBottom: '1em'}}>
                <ToggleButton toggled={lang === 'yaml'} onToggle={() => setLang(lang === 'yaml' ? 'json' : 'yaml')}>
                    YAML
                </ToggleButton>
                {buttons}
            </div>
            <div>
                <MonacoEditor
                    ref={editor}
                    key='editor'
                    defaultValue={text}
                    language={lang}
                    height='400px'
                    options={{
                        readOnly: onChange === null,
                        minimap: {enabled: false},
                        lineNumbers: 'off',
                        renderIndentGuides: false,
                        scrollBeyondLastLine: true
                    }}
                    onChange={v => {
                        if (onChange) {
                            try {
                                onChange(parse(v));
                                setError(null);
                            } catch (e) {
                                setError(e);
                            }
                        }
                    }}
                />
            </div>
            {error && (
                <div style={{paddingTop: '1em'}}>
                    <PhaseIcon value='Error' /> {error.message}
                </div>
            )}
            {onChange && (
                <div>
                    <i className='fa fa-info-circle' />{' '}
                    {lang === 'json' ? <>Full auto-completion enabled.</> : <>Basic completion for YAML. Switch to JSON for full auto-completion.</>}{' '}
                    <a href='https://argoproj.github.io/argo-workflows/ide-setup/'>Learn how to get auto-completion in your IDE.</a>
                </div>
            )}
        </>
    );
};
