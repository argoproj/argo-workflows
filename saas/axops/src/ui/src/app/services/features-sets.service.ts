import { Injectable } from '@angular/core';
import { ActivatedRouteSnapshot, CanActivate, RouterStateSnapshot } from '@angular/router';
import { Observable } from 'rxjs/Observable';
import { permissions } from '../permissions';
import { SystemService } from './system.service';

@Injectable()
export class FeaturesSetsService {

    private featuresSet: Promise<string>;

    constructor(private systemService: SystemService) {
        this.featuresSet = this.systemService.getVersion().toPromise().then(info => info.features_set || 'limited');
    }

    public async getFeaturesSet(): Promise<string> {
        return this.featuresSet;
    }

    public checkAccess(state: RouterStateSnapshot) {
        return new Observable<boolean>(observer => {
            this.getFeaturesSet().then(featuresSet => {
                let flag = true;
                permissions.forEach(p => {
                    if (p.path.indexOf('*') > -1 && state.url === p.path
                        || state.url.indexOf(p.path.split('*')[0].replace(/\/$/, '')) > -1) {
                        if (p['featuresSets']) {
                            flag = p['featuresSets'].indexOf(featuresSet) > - 1;
                        }
                    }
                });
                return observer.next(flag);
            });
        }).first().toPromise();
    }
}

@Injectable()
export class FeaturesSetsAccessControl implements CanActivate {

    constructor(private featuresSetsService: FeaturesSetsService) {
    }

    public canActivate(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<boolean> | boolean {
        return new Observable<boolean>(observer => {
            this.featuresSetsService.checkAccess(state).then((success: boolean) => {
                return observer.next(success);
            });
        });
    }
}
