export interface VolumeConfiguration {
    volume_type: string;
    filesystem: string;
}

export interface StorageClass {
    id: string;
    name: string;
    description?: string;
    parameters: {
        aws: VolumeConfiguration & { storage_provider_id: string, storage_provider_name: string };
    };
}

export interface VolumeStats {
    time: number;
    iops_read: number;
    iops_write: number;
    latency_read: number;
    latency_write: number;
    size: number;
}

export class Volume {
    id: string;
    name: string;
    anonymous: boolean;
    creator: string;
    owner: string;
    storage_provider: string;
    storage_provider_id: string;
    storage_class: string;
    storage_class_id: string;
    enabled: boolean;
    axrn: string;
    owner_id: string;
    creator_id: string;
    status: string;
    status_detail?: string;
    concurrency: number;
    resource_id: string;
    referrers?: { deployment_id: string, service_id: string, application_id: string, application_generation: string }[];
    attributes: VolumeConfiguration & { size_gb: number, usage_gb: number, free_bytes: number };
    ctime: number;
    mtime: number;
    atime: number;

    constructor() {
        this.referrers = [];
        this.attributes = {
            size_gb: undefined, usage_gb: undefined, volume_type: undefined, filesystem: undefined, free_bytes: undefined };
    }

    public clone(): Volume {
        return Object.assign(new Volume(), this);
    }

    public get parameters(): {title: string, value: string}[] {
        return [{
            title: 'storage provider name',
            value: this.storage_provider
        }, {
            title: 'volume type',
            value: this.attributes.volume_type,
        }, {
            title: 'file system',
            value: this.attributes.filesystem,
        }, {
            title: 'creator',
            value: this.creator,
        }, {
            title: 'owner',
            value: this.owner,
        }];
    }

    public get deploymentsCount() {
        return new Set(this.referrers.map(item => item.deployment_id)).size;
    }

    public get applicationsCount() {
        return new Set(this.referrers.map(item => item.application_id)).size;
    }

    public get usagePercentage() {
        return (this.attributes.usage_gb / this.attributes.size_gb * 100).toFixed();
    }

    public get storageClass(): StorageClass {
        return {
            id: this.storage_class_id,
            name: this.storage_class,
            parameters: {
                aws: Object.assign({}, this.attributes, { storage_provider_id: this.storage_provider_id, storage_provider_name: this.storage_provider })
            }
        };
    }
}
