import { Injectable } from '@angular/core';
import { ActivatedRouteSnapshot, CanActivate, RouterStateSnapshot } from '@angular/router';
import { Observable } from 'rxjs/Observable';
import { permissions } from '../permissions';

@Injectable()
export class FeaturesSetsService {

    public getFeaturesSet() {
        return new Observable<boolean>(observer => {
            return observer.next({
                'namespace': 'axsys', 'version': '1.1.0', 'cluster_id': 'dev-new-49ec3a76-70ce-11e7-a173-025000000001',
                'features_set': 'lite',
            });
        }).delay(200).first().toPromise();
    }

    public checkAccess(state: RouterStateSnapshot) {
        return new Observable<boolean>(observer => {
            this.getFeaturesSet().then(data => {
                let flag = true;
                permissions.forEach(p => {
                    if (p.path.indexOf('*') > -1 && state.url === p.path
                        || state.url.indexOf(p.path.split('*')[0].replace(/\/$/, '')) > -1) {
                        if (p['featuresSets']) {
                            flag = p['featuresSets'].indexOf(data['features_set']) > - 1;
                        }
                    }
                });
                return observer.next(flag);
            });
        }).delay(200).first().toPromise();
    }
}

@Injectable()
export class FeaturesSetsAccessControl implements CanActivate {

    constructor(private featuresSetsService: FeaturesSetsService) {
    }

    canActivate(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<boolean> | boolean {
        return new Observable<boolean>(observer => {
            this.featuresSetsService.checkAccess(state).then((success: boolean) => {
                return observer.next(success);
            });
        });
    }
}
