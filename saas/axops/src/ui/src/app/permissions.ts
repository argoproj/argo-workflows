/**
 * This is a base version of permission definition
 * The system will look at all URL being routed explicitly in /app path
 *
 * This permission mapping is very nascent version as it only looks at user groups to prevent feature access.
 * This is good for now.
 *
 * Based on my discussions with engineers in api team: The group is a very high level property.
 *
 * A 'Group' will be composed of 'Roles'. 'Roles' will be composed of 'permissions' on objects.
 * Eventually we will need to base our permissions framework on the lowest level objects - permissions
 *
 * In future admin might have the capability to define roles and groups in the system. So permissions will
 * be our fundamental level of control on the view layer.
 */
export const permissions = [
    /**
     * Login flow is open for all
     */
    { path: '/login', permission: [] },
    /**
     * Generic application workflows are open to all
     * here '*' means match routes that start with prefix (string before *)
     * and match all sub routes
     *
     * If no '*' is added to the path - We will hard check for the exact path.
     */
    { path: '/app/timeline/*', permission: [] },
    { path: '/app/policies/*', permission: ['super_admin', 'admin', 'developer'] },
    { path: '/app/service-catalog/*', permission: ['super_admin', 'admin', 'developer'] },
    { path: 'app/cashboard', permission: ['super_admin', 'admin', 'developer'] },
    { path: '/app/performance', permission: ['super_admin', 'admin', 'developer'] },
    { path: '/app/hosts', permission: ['super_admin', 'admin', 'developer'] },
    { path: '/app/metrics/*', permission: ['super_admin', 'admin', 'developer'] },
    /**
     * Administrative workflows are only open to admin user group
     */
    { path: '/app/source-control/*', permission: ['super_admin', 'admin'] },
    { path: '/app/notification/*', permission: ['super_admin', 'admin'] },
    { path: '/app/container-registry/*', permission: ['super_admin', 'admin'] },
    { path: '/app/user-management/*', permission: ['super_admin', 'admin'] },
    { path: '/app/saml/*', permission: ['super_admin', 'admin'] },
    { path: '/app/domain-management', permission: ['super_admin', 'admin'] }
];
