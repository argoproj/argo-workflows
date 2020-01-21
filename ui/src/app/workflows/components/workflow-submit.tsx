import * as models from '../../../models';
import * as React from "react";
import {Formik} from "formik";
import * as jsYaml from "js-yaml";

interface WorkflowSubmitProps {
    defaultWorkflow: models.Workflow;
}

interface WorkflowSubmitState {
    wf: models.Workflow;
    wfString: string;
}

export class WorkflowSubmit extends React.Component<WorkflowSubmitProps, WorkflowSubmitState> {
    constructor(props: WorkflowSubmitProps) {
        super(props);
        this.state = {
            wf: this.props.defaultWorkflow,
            wfString: jsYaml.dump(this.props.defaultWorkflow),
        }
    }

    public render() {
        return (
            <div>
                <Formik
                    initialValues={{wf: this.state.wf, wfString: this.state.wfString}}
                    onSubmit={(values, {setSubmitting}) => {
                        setTimeout(() => {
                            alert(JSON.stringify(values, null, 2));
                            setSubmitting(false);
                        }, 400);
                    }}
                >
                    {({
                          values,
                          errors,
                          touched,
                          handleChange,
                          handleBlur,
                          handleSubmit,
                          isSubmitting,
                          setFieldValue
                          /* and other goodies */
                      }) => (
                        <form onSubmit={handleSubmit}>
                            <div className='white-box editable-panel'>
                                <h4>Submit New Workflow</h4>
                                <button type="submit" className='argo-button argo-button--base' disabled={isSubmitting}>
                                    Submit
                                </button>
                                <textarea
                                    name={"wfString"}
                                    className='yaml'
                                    value={values.wfString}
                                    onChange={e => {
                                        handleChange(e);
                                    }}
                                    onBlur={e => {
                                        handleBlur(e);
                                        try {
                                            setFieldValue("wf", jsYaml.load(e.currentTarget.value))
                                        } catch (e) {
                                            console.log("INVALID YAML")
                                        }
                                    }}
                                    onFocus={e => (e.currentTarget.style.height = e.currentTarget.scrollHeight + 'px')}
                                    autoFocus={true}
                                />

                                <div className='white-box__details'>
                                    {console.log(values.wf)}
                                    {values.wf && values.wf.spec && values.wf.spec.arguments &&
                                    values.wf.spec.arguments.parameters &&
                                    values.wf.spec.arguments.parameters.map(function (param: models.Parameter, index: number) {
                                        if (param != null) {
                                            return (
                                                <div className='argo-form-row'>
                                                    <label className='argo-label-placeholder'
                                                           htmlFor={"wf.spec.arguments.parameters[" + index + "].value"}>
                                                        {param.name}
                                                    </label>
                                                    <input className='argo-field'
                                                           key={"wf.spec.arguments.parameters[" + index + "].value"}
                                                           name={"wf.spec.arguments.parameters[" + index + "].value"}
                                                           type={"text"}
                                                           value={param.value} onChange={handleChange} onBlur={e => {
                                                        handleBlur(e);
                                                        setFieldValue("wfString", jsYaml.dump(values.wf))
                                                    }}/>
                                                </div>
                                            )

                                        }
                                    })}
                                </div>
                            </div>
                        </form>
                    )}
                </Formik>
            </div>
        )
    }


    // private get appContext(): AppContext {
    //     return this.context as AppContext;
    // }

}