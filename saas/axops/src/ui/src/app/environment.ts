import { enableDebugTools, disableDebugTools } from '@angular/platform-browser';
import { enableProdMode, ApplicationRef } from '@angular/core';

let PROVIDERS: any[] = [
    // common env directives
];

// Angular debug tools in the dev console
// https://github.com/angular/angular/blob/86405345b781a9dc2438c0fbe3e9409245647019/TOOLS_JS.md

/* tslint:disable */
let decorateModuleRefInternal = function identity<T>(value: T): T { return value; };
/* tslint:enable */

if ('production' === ENV) {
    // Production
    disableDebugTools();
    enableProdMode();

    PROVIDERS = [
        ...PROVIDERS,
    ];

} else {
    decorateModuleRefInternal = (modRef: any) => {
        const appRef = modRef.injector.get(ApplicationRef);
        const cmpRef = appRef.components[0];

        let ng = (<any> window).ng;
        enableDebugTools(cmpRef);
        (<any> window).ng.probe = ng.probe;
        (<any> window).ng.coreTokens = ng.coreTokens;
        return modRef;
    };

    // Development
    PROVIDERS = [
        ...PROVIDERS,
    ];

}

export const decorateModuleRef = decorateModuleRefInternal;

export const ENV_PROVIDERS = [
    ...PROVIDERS,
];
