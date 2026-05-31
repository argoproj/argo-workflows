import '@testing-library/jest-dom';

// react-router v7's Node build references TextEncoder/TextDecoder, which jsdom does not
// provide by default. Polyfill them from Node's util module before any test code runs.
import {TextDecoder, TextEncoder} from 'util';

if (typeof globalThis.TextEncoder === 'undefined') {
    globalThis.TextEncoder = TextEncoder as typeof globalThis.TextEncoder;
}
if (typeof globalThis.TextDecoder === 'undefined') {
    globalThis.TextDecoder = TextDecoder as typeof globalThis.TextDecoder;
}
