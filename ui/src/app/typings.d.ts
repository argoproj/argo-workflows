declare let SYSTEM_INFO: {version: string};

declare module 'swagger-ui-react/swagger-ui.css';

// TODO: remove this once we've updated to TS v5.1+ (c.f. https://github.com/microsoft/TypeScript/issues/49231#issuecomment-1137251612)
declare namespace Intl {
    type Key = 'calendar' | 'collation' | 'currency' | 'numberingSystem' | 'timeZone' | 'unit';

    function supportedValuesOf(input: Key): string[];
}
