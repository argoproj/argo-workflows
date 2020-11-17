import {languages} from 'monaco-editor/esm/vs/editor/editor.api';
import * as React from 'react';
import {useEffect} from 'react';
import MonacoEditor from 'react-monaco-editor';
import {uiUrl} from '../../base';
import {parse, stringify} from './resource';

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
        if (props.type) {
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
    });

    return (
        <>
            <MonacoEditor
                key='editor'
                value={stringify(props.value, language)}
                language={language}
                height='600px'
                onChange={value => props.onChange && props.onChange(parse(value))}
                options={{
                    readOnly: props.onChange === null,
                    extraEditorClassName: 'resource',
                    minimap: {enabled: false},
                    lineNumbers: 'off',
                    renderIndentGuides: false
                }}
            />
            {props.onChange && (
                <div style={{marginTop: '1em'}}>
                    <i className='fa fa-info-circle'/>{' '}
                    {props.language === 'json' ? <>Full auto-completion enabled.</> : <>Basic completion for YAML.
                        Switch to JSON for full auto-completion.</>}{' '}
                    <a href='https://argoproj.github.io/argo/ide-setup/'>Learn how to get auto-completion in your
                        IDE.</a>
                </div>
            )}
        </>
    );
};
