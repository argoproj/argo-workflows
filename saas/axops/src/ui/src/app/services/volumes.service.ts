import { Injectable } from '@angular/core';
import { Http, URLSearchParams } from '@angular/http';
import { Volume, StorageClass, VolumeStats } from '../model';
import * as moment from 'moment';

import { AxHeaders } from './headers';
import { DeploymentsService } from './deployments.service';

@Injectable()
export class VolumesService {

    constructor(private deploymentsService: DeploymentsService, private http: Http) {}

    public async getVolumes(params: { named?: boolean, deployment_id?: string } = {}, hideLoader: boolean = true): Promise<Volume[]> {
        let search = new URLSearchParams();
        if (typeof params.named === 'boolean') {
            search.set('anonymous', String(!params.named));
        }

        if (params.deployment_id) {
            search.set('deployment_id', params.deployment_id);
        }

        return this.http.get('v1/storage/volumes' , {search: search, headers: new AxHeaders({noLoader: hideLoader}) }).toPromise()
            .then(res => <Volume[]> res.json().data.map(item => this.deserializeVolume(item)));
    }

    public getStorageClasses(hideLoader: boolean = true): Promise<StorageClass[]> {
        return this.http.get('v1/storage/classes', {headers: new AxHeaders({noLoader: hideLoader}) }).toPromise().then(res => res.json().data);
    }

    public createVolume(name: string, sizeGb: number, storageClass: StorageClass): Promise<Volume> {
        return this.http.post('v1/storage/volumes', this.getVolumeUpdateData(name, sizeGb, storageClass)).toPromise().then(res => res.json());
    }

    public updateVolume(volume: Volume): Promise<Volume> {
        return this.http.put(
            `v1/storage/volumes/${volume.id}`,
            this.getVolumeUpdateData(volume.name, volume.attributes.size_gb, volume.storageClass
        )).toPromise().then(res => this.deserializeVolume(res.json()));
    }

    public async getVolumeById(id: string): Promise<Volume> {
        return this.http.get(`v1/storage/volumes/${id}`).toPromise().then(res => this.deserializeVolume(res.json()));
    }

    public async deleteVolume(id: string) {
        return this.http.delete(`v1/storage/volumes/${id}`).toPromise();
    }

    public async getStats(id: string): Promise<VolumeStats[]> {
        return this.http.get(`v1/storage/volumes/${id}`).toPromise().then(res => res.json());
    }

    public async getChartStats(id: string,
                               type: 'readops' | 'writeops' | 'readtot' | 'writetot' | 'readsizetot' | 'writesizetot' | 'readsizeavg' | 'writesizeavg',
                               interval: number = 3600,
                               start?: moment.Moment,
                               end?: moment.Moment, hideLoader: boolean = true): Promise<any[]> {
        let search = new URLSearchParams();

        search.set('interval', interval.toString());

        if (type) {
            search.set('type', type);
        }

        if (start) {
            search.set('min_time', start.unix().toString());
        }

        if (end) {
            search.set('max_time', end.unix().toString());
        }

        return this.http.get(`v1/storage/volumes/${id}/stats`,
            {search: search, headers: new AxHeaders({noLoader: hideLoader, noErrorHandling: true}) }).toPromise().then(res => res.json()).catch(e => []);
    }

    public getVolumeUpdateData(name: string, sizeGb: number, storageClass: StorageClass) {
        return {
            name : name,
            storage_provider: storageClass.parameters.aws.storage_provider_name,
            storage_provider_id : storageClass.parameters.aws.storage_provider_id,
            storage_class: storageClass.name,
            storage_class_id : storageClass.id,
            attributes : {
                volume_type : storageClass.parameters.aws.volume_type,
                filesystem : storageClass.parameters.aws.filesystem,
                size_gb : sizeGb
            }
        };
    }

    private deserializeVolume(data: any) {
        let volume = new Volume();
        Object.assign(volume, data);
        if (typeof volume.attributes.size_gb === 'string') {
            volume.attributes.size_gb = parseInt(volume.attributes.size_gb as string, 10);
        }
        let bytesInGb = 1024.0 * 1024.0 * 1024.0;
        if (volume.attributes.free_bytes === undefined) {
            volume.attributes.free_bytes = volume.attributes.size_gb * bytesInGb;
        };
        volume.attributes.usage_gb = +(volume.attributes.size_gb - volume.attributes.free_bytes / bytesInGb).toFixed(2);
        volume.name = volume.name || volume.resource_id;
        if (volume.referrers) {
            volume.referrers.forEach(ref => {
                ref.deployment_id = ref.service_id;
            });
        }
        return volume;
    }
}
