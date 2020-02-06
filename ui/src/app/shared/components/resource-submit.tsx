import {Formik} from 'formik';
import * as jsYaml from 'js-yaml';
import * as React from 'react';
import * as models from '../../../models';

interface ResourceSubmitProps<T> {
    defaultResource: T;
    resourceName: string;
    onSubmit: (value: T) => Promise<void>;
}

interface ResourceSubmitState {
    invalid: boolean;
    error?: any;
}

export class ResourceSubmit<T> extends React.Component<ResourceSubmitProps<T>, ResourceSubmitState> {
    constructor(props: ResourceSubmitProps<T>) {
        super(props);
        this.state = {invalid: false};
    }

    public render() {
        return (
            <div>
                <Formik
                    initialValues={{resource: this.props.defaultResource, resourceString: jsYaml.dump(this.props.defaultResource)}}
                    onSubmit={(values, {setSubmitting}) => {
                        this.props
                            .onSubmit(values.resource)
                            .then(_ => setSubmitting(false))
                            .catch(error => {
                                this.setState({error});
                                setSubmitting(false);
                            });
                    }}>
                    {(formikApi: any) => (
                        <form onSubmit={formikApi.handleSubmit}>
                            <div className='white-box editable-panel'>
                                <h4>Submit New {this.props.resourceName}</h4>
                                <button type='submit' className='argo-button argo-button--base' disabled={formikApi.isSubmitting || this.state.invalid}>
                                    Submit
                                </button>
                                {this.state.error && (
                                    <p>
                                        <i className='fa fa-exclamation-triangle status-icon--failed' />
                                        {this.state.error.response && this.state.error.response.body && this.state.error.response.body.message
                                            ? this.state.error.response.body.message
                                            : this.state.error.message}
                                    </p>
                                )}
                                <textarea
                                    name={'resourceString'}
                                    className='yaml'
                                    value={formikApi.values.resourceString}
                                    onChange={e => {
                                        formikApi.handleChange(e);
                                    }}
                                    onBlur={e => {
                                        formikApi.handleBlur(e);
                                        try {
                                            formikApi.setFieldValue('resource', jsYaml.load(e.currentTarget.value));
                                            this.setState({
                                                error: undefined,
                                                invalid: false
                                            });
                                        } catch (e) {
                                            this.setState({
                                                error: {
                                                    name: this.props.resourceName + ' is invalid',
                                                    message: this.props.resourceName + ' is invalid' + (e.reason ? ': ' + e.reason : '')
                                                },
                                                invalid: true
                                            });
                                        }
                                    }}
                                    onFocus={e => (e.currentTarget.style.height = e.currentTarget.scrollHeight + 'px')}
                                    autoFocus={true}
                                />

                                {/* Workflow-level parameters*/}
                                {this.props.resourceName === 'Workflow' &&
                                    formikApi.values.resource &&
                                    formikApi.values.resource.spec &&
                                    formikApi.values.resource.spec.arguments &&
                                    formikApi.values.resource.spec.arguments.parameters &&
                                    this.renderParameterFields('Workflow Parameters', 'resource.spec.arguments', formikApi.values.resource.spec.arguments.parameters, formikApi)}
                            </div>
                        </form>
                    )}
                </Formik>
            </div>
        );
    }

    private renderParameterFields(sectionTitle: string, path: string, parameters: models.Parameter[], formikApi: any): JSX.Element {
        return (
            <div className='white-box__details' style={{paddingTop: '50px'}}>
                <h5>{sectionTitle}</h5>
                {parameters.map((param: models.Parameter, index: number) => {
                    if (param != null) {
                        return (
                            <div className='argo-form-row'>
                                <label className='argo-label-placeholder' htmlFor={path + '.parameters[' + index + '].value'}>
                                    {param.name}
                                </label>
                                <input
                                    className='argo-field'
                                    key={path + '.parameters[' + index + '].value'}
                                    name={path + '.parameters[' + index + '].value'}
                                    type={'text'}
                                    value={param.value}
                                    onChange={formikApi.handleChange}
                                    onBlur={e => {
                                        formikApi.handleBlur(e);
                                        formikApi.setFieldValue('resourceString', jsYaml.dump(formikApi.values.resource));
                                    }}
                                />
                            </div>
                        );
                    }
                })}
            </div>
        );
    }
}
