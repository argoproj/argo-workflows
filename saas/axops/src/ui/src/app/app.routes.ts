import * as _ from 'lodash';
import { HasNoSession, UserAccessControl, FeaturesSetsAccessControl } from './services';
import { routes } from './routes';

const accessChecks = [UserAccessControl, FeaturesSetsAccessControl];
/**
 * All routes passing through this method get applied with accessChecks
 * As of now accessChecks comprise of session check and check for permissions.
 *
 * This method recurses into all child routes and applies the same accessChecks.
 */
function addAccessControlChecks(route) {
    if (route.canActivate) {
        route.canActivate = _.concat(route.canActivate, accessChecks);
    } else {
        route.canActivate = accessChecks;
    }
    if (route.children && route.children.length > 0) {
        for (let i = 0; i < route.children.length; i++) {
            addAccessControlChecks(route.children[i]);
        }
    }
}

/**
 * Decorate the route definitions with permission controls.
 * We have added hook for all /app/* routes to have canActivate filters applied
 *
 * /app route has only session check added ('HasSession')
 * All child routes have accessChecks checks applied via addAccessControlChecks
 */
export function decorateRouteDefs(routeDefs: Array<any>, forceAddGuard?: boolean) {
    for (let i = 0; i < routeDefs.length; i++) {
        if (routeDefs[i].path === 'app' || forceAddGuard) {
            routeDefs[i].canActivate = [];

            if (routeDefs[i].children && routeDefs[i].children.length > 0 || forceAddGuard) {
                addAccessControlChecks(routeDefs[i]);
            }
        } else if (routeDefs[i].path === 'login/:fwd' || routeDefs[i].path === 'login' ) {
            routeDefs[i].canActivate = [HasNoSession];
        }
    }

    return routeDefs;
}
export const ROUTERS = decorateRouteDefs(routes);
