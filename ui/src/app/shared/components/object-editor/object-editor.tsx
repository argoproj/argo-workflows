import {languages} from 'monaco-editor/esm/vs/editor/editor.api';
import * as React from 'react';
import {createRef, useEffect} from 'react';
import MonacoEditor from 'react-monaco-editor';
import {uiUrl} from '../../base';
import {parse, stringify} from '../object-parser';

interface Props<T> {
    language?: string;
    type: string;
    value: T;
    onChange?: (value: T) => void;
    onError?: (error: Error) => void;
}

export const ObjectEditor = <T extends any>(props: Props<T>) => {
    const language = props.language || 'yaml';

    useEffect(() => {
        if (props.type && language === 'json') {
            const uri = uiUrl('assets/openapi-spec/swagger.json');
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
                                    $id: 'http://workflows.argoproj.io/' + props.type + '.json',
                                    $ref: '#/definitions/' + props.type,
                                    $schema: 'http://json-schema.org/draft-07/schema',
                                    definitions: swagger.definitions
                                }
                            }
                        ]
                    });
                })
                .catch(error => props.onError(error));
        }
    }, [language]);

    const editor = createRef<MonacoEditor>();

    return (
        <div onBlur={() => props.onChange && props.onChange(parse(editor.current.editor.getModel().getValue()))}>
            <MonacoEditor
                ref={editor}
                key='editor'
                value={stringify(props.value, language)}
                language={language}
                height='600px'
                options={{
                    readOnly: props.onChange === null,
                    minimap: {enabled: false},
                    lineNumbers: 'off',
                    renderIndentGuides: false
                }}
            />
            {props.onChange && (
                <p>
                    <i className='fa fa-info-circle' />{' '}
                    {props.language === 'json' ? <>Full auto-completion enabled.</> : <>Basic completion for YAML. Switch to JSON for full auto-completion.</>}{' '}
                    <a href='https://argoproj.github.io/argo/ide-setup/'>Learn how to get auto-completion in your IDE.</a>
                </p>
            )}
        </div>
    );
};
