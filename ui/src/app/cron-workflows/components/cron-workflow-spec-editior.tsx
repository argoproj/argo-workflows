import {Checkbox, Select} from 'argo-ui';
import * as React from 'react';
import {ConcurrencyPolicy, CronWorkflowSpec} from '../../../models';
import {NumberInput} from '../../shared/components/number-input';
import {TextInput} from '../../shared/components/text-input';
import {ScheduleValidator} from './schedule-validator';

export function CronWorkflowSpecEditor({onChange, spec}: {spec: CronWorkflowSpec; onChange: (spec: CronWorkflowSpec) => void}) {
    return (
        <div className='white-box'>
            <div className='white-box__details'>
                <div className='row white-box__details-row'>
                    <div className='columns small-3'>Schedule</div>
                    <div className='columns small-9'>
                        <TextInput value={spec.schedule} onChange={schedule => onChange({...spec, schedule})} />
                        <ScheduleValidator schedule={spec.schedule} />
                    </div>
                </div>
                <div className='row white-box__details-row'>
                    <div className='columns small-3'>Timezone</div>
                    <div className='columns small-9'>
                        <TextInput value={spec.timezone} onChange={timezone => onChange({...spec, timezone})} />
                    </div>
                </div>
                <div className='row white-box__details-row'>
                    <div className='columns small-3'>Concurrency Policy</div>
                    <div className='columns small-9' style={{lineHeight: '30px'}}>
                        <Select
                            placeholder='Select concurrency policy'
                            options={['Allow', 'Forbid', 'Replace']}
                            value={spec.concurrencyPolicy}
                            onChange={x =>
                                onChange({
                                    ...spec,
                                    concurrencyPolicy: x.value as ConcurrencyPolicy
                                })
                            }
                        />
                    </div>
                </div>
                <div className='row white-box__details-row'>
                    <div className='columns small-3'>Starting Deadline Seconds</div>
                    <div className='columns small-9'>
                        <NumberInput
                            value={spec.startingDeadlineSeconds}
                            onChange={startingDeadlineSeconds =>
                                onChange({
                                    ...spec,
                                    startingDeadlineSeconds
                                })
                            }
                        />
                    </div>
                </div>
                <div className='row white-box__details-row'>
                    <div className='columns small-3'>Successful Jobs History Limit</div>
                    <div className='columns small-9'>
                        <NumberInput
                            value={spec.successfulJobsHistoryLimit}
                            onChange={successfulJobsHistoryLimit =>
                                onChange({
                                    ...spec,
                                    successfulJobsHistoryLimit
                                })
                            }
                        />
                    </div>
                </div>
                <div className='row white-box__details-row'>
                    <div className='columns small-3'>Failed Jobs History Limit</div>
                    <div className='columns small-9'>
                        <NumberInput
                            value={spec.failedJobsHistoryLimit}
                            onChange={failedJobsHistoryLimit =>
                                onChange({
                                    ...spec,
                                    failedJobsHistoryLimit
                                })
                            }
                        />
                    </div>
                </div>
                <div className='row white-box__details-row'>
                    <div className='columns small-3'>Suspended</div>
                    <div className='columns small-9'>
                        <Checkbox checked={spec.suspend} onChange={suspend => onChange({...spec, suspend})} />
                    </div>
                </div>
            </div>
        </div>
    );
}
