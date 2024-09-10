import * as React from 'react';
import {useState} from 'react';

import {DisplayWorkflowTime} from '../workflow-node-info/workflow-node-info';
import {TIMESTAMP_KEYS} from '../../../shared/use-timestamp';
import {Button} from '../../../shared/components/button';
import {GetUserInfoResponse, kubernetes} from '../../../../models';
import {services} from '../../../shared/services';
import {ErrorNotice} from '../../../shared/components/error-notice';

interface SuspendApprovalProps {
    nodeId: string;
    wfStartTime: kubernetes.Time;
    approverStatus: Map<string, boolean>;
    setApproverStatus: (key: string, value: boolean) => void;
}

export function SuspendApprovals(props: SuspendApprovalProps) {
    const [approverStatus, setApproverStati] = useState<Map<string, boolean>>(props.approverStatus);
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
        setApproverStati(previous => {
            const newMap = new Map(previous);
            newMap.set(key, value);
            return newMap;
        });
        props.setApproverStatus(key, value);
    }

    function renderApprovalRows(status: Map<string, boolean>) {
        let index = -1;
        const body: {id: number; approver: string; icon: JSX.Element; name: JSX.Element; status: JSX.Element; time: JSX.Element; action: JSX.Element}[] = [];
        status.forEach((approverStatus, approver) => {
            index += 1;
            body.push({
                id: index,
                approver: approver,
                icon: (
                    <div className='columns small-1'>
                        <i className='fa fa-solid fa-user-tie workflows-list__status--approval'></i>
                    </div>
                ),
                name: (
                    <div className='columns small-4'>
                        <span>{approver}</span>
                    </div>
                ),
                status: (
                    <div className='columns small-2'>
                        <span>
                            {approverStatus ? (
                                <i className='fa fa-check-circle status-icon--success' aria-hidden='true'></i>
                            ) : (
                                <i className='fa fa-circle-notch status-icon--running status-icon--spin' aria-hidden='true'></i>
                            )}
                        </span>
                        <span>{approverStatus ? ' approved' : ' pending approval'}</span>
                    </div>
                ),
                time: (
                    <div className='columns small-3'>
                        <DisplayWorkflowTime date={props.wfStartTime} timestampKey={TIMESTAMP_KEYS['WORKFLOW_NODE_STARTED']} />
                    </div>
                ),
                action: (
                    <div className='columns small-2'>
                        <Button icon='check' disabled={approver !== userInfo?.email || approverStatus} onClick={() => setApproverStatus(approver)}>
                            Approve
                        </Button>
                    </div>
                )
            });
        });

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

    function renderFields(status: Map<string, boolean>) {
        return renderApprovalRows(status);
    }

    function renderInputContentIfApplicable() {
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
