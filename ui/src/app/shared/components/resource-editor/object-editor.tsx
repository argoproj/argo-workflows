import * as React from 'react';
import MonacoEditor from 'react-monaco-editor';
import {uiUrl} from '../../base';

import {languages} from 'monaco-editor/esm/vs/editor/editor.api';
import {parse, stringify} from './resource';

interface Props<T> {
    language?: string;
    type: string;
    value: T;
    onChange?: (value: T) => void;
    onError?: (error: Error) => void;
}

interface State {
    value: string;
}

export class ObjectEditor<T> extends React.Component<Props<T>, State> {
    private get language() {
        return this.props.language || 'yaml';
    }
    constructor(props: Readonly<Props<T>>) {
        super(props);
        this.state = {value: stringify(this.props.value, this.props.language || 'yaml')};
    }

    public componentDidMount() {
        if (this.props.type) {
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
                                    $id: 'http://workflows.argoproj.io/' + this.props.type + '.json',
                                    $ref: '#/definitions/' + this.props.type,
                                    $schema: 'http://json-schema.org/draft-07/schema',
                                    definitions: swagger.definitions
                                }
                            }
                        ]
                    });
                })
                .catch(error => this.props.onError(error));
        }
    }

    public componentDidUpdate(prevProps: Props<T>) {
        if (prevProps.value !== this.props.value || prevProps.language !== this.props.language) {
            this.setState(() => ({value: stringify(this.props.value, this.language)}));
        }
    }

    public render() {
        return (
            <>
                <MonacoEditor
                    key='editor'
                    value={this.state.value}
                    language={this.language}
                    height='600px'
                    onChange={value => this.setState({value}, () => this.props.onChange && this.props.onChange(parse(this.state.value)))}
                    options={{
                        readOnly: this.props.onChange === null,
                        extraEditorClassName: 'resource',
                        minimap: {enabled: false},
                        lineNumbers: 'off',
                        renderIndentGuides: false
                    }}
                />
                <div style={{marginTop: '1em'}}>
                    <i className='fa fa-info-circle' />{' '}
                    {this.props.language === 'json' ? <>Full auto-completion enabled.</> : <>Basic completion for YAML. Switch to JSON for full auto-completion.</>}{' '}
                    <a href='https://argoproj.github.io/argo/ide-setup/'>Learn how to get auto-completion in your IDE.</a>
                </div>
            </>
        );
    }
}
