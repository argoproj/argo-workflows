export enum SystemRequestType {
    UserInvite = 1,
    UserConfirm = 2,
    PassReset = 3,
}

export class SystemRequest {
    id: string;
    user_id: string;
    user_name: string;
    target: string;
    type: SystemRequestType;
    expiry: number;
    data: any;
}
