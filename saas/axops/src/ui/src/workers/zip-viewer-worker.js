'use strict';

this.addEventListener('fetch', function(event) {
    if (new URL(event.request.url).pathname.indexOf('/assets/workers/zip-viewer/') === 0) {
        event.respondWith(caches.match(event.request));
    }  
});
