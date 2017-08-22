import { Task  } from '../../model';

export interface Job {
    status: number;
    elements: Task[];
}

export interface StepInfo {
    name: string;
    value: Task;
    children: StepInfo[];
    isSkipped: boolean;
    isSucceeded: boolean;
    isRunning: boolean;
    isFailed: boolean;
    isNotStarted: boolean;
    isCancelled: boolean;
    stepLayer: number;
    lastInLayer: boolean;
    firstInLayer: boolean;
}
