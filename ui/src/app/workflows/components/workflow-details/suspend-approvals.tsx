import * as React from 'react';
import {useState} from 'react';

import {DisplayWorkflowTime} from '../workflow-node-info/workflow-node-info';
import {TIMESTAMP_KEYS} from '../../../shared/use-timestamp';
import {Button} from '../../../shared/components/button';
import {ApproverStatus, GetUserInfoResponse, kubernetes} from '../../../../models';
import {services} from '../../../shared/services';
import {ErrorNotice} from '../../../shared/components/error-notice';

interface SuspendApprovalProps {
    nodeId: string;
    wfStartTime: kubernetes.Time;
    approverStatus: ApproverStatus[];
    setApproverStatus: (key: string, value: boolean) => void;
}

export function SuspendApprovals(props: SuspendApprovalProps) {
    const [approverStatus, setApproverStati] = useState<ApproverStatus[]>(props.approverStatus);
    const [error, setError] = useState<Error>(null);
    const [userInfo, setUserInfo] = useState<GetUserInfoResponse>();

    React.useEffect(() => {
        (async function getUserInfoWrapper() {
            try {
                const newUserInfo = await services.info.getUserInfo(); // TODO verify this works, idk how
                setUserInfo(newUserInfo);
                setError(null);
            } catch (newError) {
                setError(newError);
            }
        })();
    }, []);

    function setApproverStatus(key: string, value: boolean = true) {
        props.setApproverStatus(key, value);
        setApproverStati(previous => {
            return previous.map(param => {
                if (param.approver === key) {
                    param.approvalStatus = value;
                }
                return param;
            });
        });
    }

    function renderApprovalRows(status: ApproverStatus[]) {
        const body = status.map((approver, index) => {
            console.log(approver.approver);
            console.log(userInfo?.email);
            return {
                id: index,
                approver: approver.approver,
                icon: (
                    <div className='columns small-1'>
                        <i className='fa fa-solid fa-user-tie workflows-list__status--approval'></i>
                    </div>
                ),
                name: (
                    <div className='columns small-4'>
                        <span>{approver.approver}</span>
                    </div>
                ),
                status: (
                    <div className='columns small-2'>
                        <span>
                            {approver.approvalStatus ? (
                                <i className='fa fa-check-circle status-icon--success' aria-hidden='true'></i>
                            ) : (
                                <i className='fa fa-circle-notch status-icon--running status-icon--spin' aria-hidden='true'></i>
                            )}
                        </span>
                        <span>{approver.approvalStatus ? 'approved' : ' pending approval'}</span>
                    </div>
                ),
                time: (
                    <div className='columns small-3'>
                        <DisplayWorkflowTime date={props.wfStartTime} timestampKey={TIMESTAMP_KEYS['WORKFLOW_NODE_STARTED']} />
                    </div>
                ),
                action: (
                    <div className='columns small-2'>
                        <Button icon='check' disabled={approver.approver !== userInfo?.email && !approver.approvalStatus} onClick={() => setApproverStatus(approver.approver)}>
                            Approve
                        </Button>
                    </div>
                )
            };
        });

        // const rows = header.concat(body);

        console.log('BODY');
        console.log(body);

        return (
            <React.Fragment>
                <ErrorNotice error={error} />
                <br />
                <div className='argo-table-list'>
                    <div className='row argo-table-list__head'>
                        <div className='columns small-1 workflows-list__status'></div>
                        <div className='row small-12'>
                            <div className='columns small-1'></div>
                            <div className='columns small-4'>APPROVER</div>
                            <div className='columns small-2'>STATUS</div>
                            <div className='columns small-3'>TIME</div>
                            <div className='columns small-2'>ACTION</div>
                        </div>
                    </div>
                    {body.map(row => (
                        <React.Fragment key={row.id}>
                            <div className='workflows-list__row-container'>
                                <div className='row argo-table-list__row'>
                                    <div className={'small-12 row approval-list' + (row.approver === userInfo?.email ? '__current' : '')}>
                                        {row.icon}
                                        {row.name}
                                        {row.status}
                                        {row.time}
                                        {row.action}
                                    </div>
                                </div>
                            </div>
                        </React.Fragment>
                    ))}
                </div>
            </React.Fragment>
        );
    }

    function renderFields(status: ApproverStatus[]) {
        return renderApprovalRows(status);
    }

    function renderInputContentIfApplicable() {
        // if (parameters.length === 0) {
        //     return <React.Fragment />;
        // }
        return (
            <React.Fragment>
                <h2>Approve</h2>
                {renderFields(approverStatus)}
                <br />
            </React.Fragment>
        );
    }

    return (
        <div>
            {renderInputContentIfApplicable()}
            <br />
            Are you sure you want to resume node {props.nodeId} ?
        </div>
    );
}
