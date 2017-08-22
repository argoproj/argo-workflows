export class Artifact {
    archive_mode: string = '';
    artifact_id: string = '';
    artifact_type: string = '';
    ax_time: number = 0;
    ax_uuid: string = '';
    ax_week: number = 0;
    checksum: string = '';
    compression_mode: string = '';
    deleted: number = 0;
    deleted_by: string = '';
    deleted_date: number = 0;
    description: string= '';
    excludes: string = '';
    full_path: string = '';
    inline_storage: string = '';
    is_alias: number = 0;
    meta: any;  // json string
    name: string = '';
    num_byte: number = 0;
    num_dir: number = 0;
    num_file: number = 0;
    num_other: number = 0;
    num_skip: number = 0;
    num_skip_byte: number = 0;
    num_symlink: number = 0;
    relative_path: string = '';
    retention_tags: string = '';
    service_instance_id: string = '';
    source_artifact_id: string = '';
    src_name: string = '';
    src_path: string = '';
    storage_method: string = '';
    storage_path: ArtifactPath = new ArtifactPath();
    stored_byte: number = 0;
    structure_path: ArtifactPath = new ArtifactPath();
    symlink_mode: string = '';
    tags: string[] = [];
    third_party: string = '';
    timestamp: number = 0;
    workflow_id: string = '';
    md5: string = '';
    alias_of: string = '';

    constructor(data?) {
        if (data && typeof data === 'object') {
            Object.assign(this, data);
        }
    }

    get alias(): string {
        return `${this.full_path}: ${this.name}`;
    }
}

export const ARTIFACT_TYPES = {
    AX_LOG: 'ax-log',
    AX_LOG_EXTERNAL: 'ax-log-external',
    USER_LOG: 'user-log',
    INTERNAL: 'internal',
    EXPORTED: 'exported'
};

export class ArtifactNums {
    'ax-log-current-nums': number = 0;
    'ax-log-original-nums': number = 0;
    'current-nums': number = 0;
    'exported-current-nums': number = 0;
    'exported-original-nums': number = 0;
    'internal-current-nums': number = 0;
    'internal-original-nums': number = 0;
    'original-nums': number = 0;
    'user-log-current-nums': number = 0;
    'user-log-original-nums': number = 0;
}

export class ArtifactSize {
    'ax-log-size': number = 0;
    'ax-log-stored-size': number = 0;
    'exported-size': number = 0;
    'exported-stored-size': number = 0;
    'internal-size': number = 0;
    'internal-stored-size': number = 0;
    'total-size': number = 0;
    'total-stored-size': number = 0;
    'user-log-size': number = 0;
    'user-log-stored-size': number = 0;
}

export class ArtifactsUsage {
    artifact_nums: ArtifactNums;
    artifact_size: ArtifactSize;
}

export class RetentionPolicy {
    description: string = '';
    name: string = '';
    policy: number = 0;
    total_number: number = 0;
    total_real_size: number = 0;
    total_size: number = 0;

    constructor(data?) {
        if (typeof data === 'object') {
            for (let key in data) {
                if (data.hasOwnProperty(key)) {
                    this[key] = data[key];
                }
            }
        }
    }
}

export class ArtifactPath {
    bucket: string = '';
    key: string = '';
}

export const ArtifactFieldNames = {
};
