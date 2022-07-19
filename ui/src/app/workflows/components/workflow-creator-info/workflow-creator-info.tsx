import * as React from 'react';
import * as models from '../../../../models';
import {labels} from '../../../../models';

require('./workflow-creator-info.scss');

interface WorkflowCreatorInfoProps {
    workflow: models.Workflow;
    onChange: (key: string, value: string) => void;
}

export class WorkflowCreatorInfo extends React.Component<WorkflowCreatorInfoProps, {}> {
    constructor(props: WorkflowCreatorInfoProps) {
        super(props);
    }

    public render() {
        const w = this.props.workflow;
        const creatorLabels = [];
        if (w.metadata.labels) {
            const creatorInfoMap = new Map<string, [string, string]>([
                ['Name', [labels.creator, w.metadata.labels[labels.creator]]],
                ['Email', [labels.creatorEmail, w.metadata.labels[labels.creatorEmail]]],
                ['Preferred username', [labels.creatorPreferredUsername, w.metadata.labels[labels.creatorPreferredUsername]]]
            ]);
            creatorInfoMap.forEach((value: [string, string], key: string) => {
                const [searchKey, searchValue] = value;
                if (searchValue !== '' && searchValue !== undefined) {
                    creatorLabels.push(
                        <div
                            title={`List workflows created by ${key}=${searchValue}`}
                            className='tag'
                            key={`${w.metadata.uid}-${key}`}
                            onClick={async e => {
                                e.preventDefault();
                                this.props.onChange(searchKey, searchValue);
                            }}>
                            <div className='key'>{key}</div>
                            <div className='value'>{searchValue}</div>
                        </div>
                    );
                }
            });
        } else {
            creatorLabels.push(<div key={`${w.metadata.uid}- `}> No creator information </div>);
        }
        return <div className='wf-row-creator-labels'>{creatorLabels}</div>;
    }
}
