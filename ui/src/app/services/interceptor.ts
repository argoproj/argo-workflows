import { Injectable } from '@angular/core';
import { HttpInterceptor, HttpHandler, HttpRequest, HttpEvent, HttpResponse } from '@angular/common/http';

import { Observable } from 'rxjs/Observable';
import 'rxjs/add/operator/do';
import { LoaderService } from './loader.service';

@Injectable()
export class Interceptor implements HttpInterceptor {


  constructor(private loaderService: LoaderService) {}

  public intercept(
    req: HttpRequest<any>,
    next: HttpHandler
  ): Observable<HttpEvent<any>> {
    const showLoader = req.headers.get('noLoader') !== 'true';
    if (showLoader) {
      this.loaderService.startBackgroudJob();
    }
    return next.handle(req).finally(() => {
      if (showLoader) {
        this.loaderService.completeBackgroudJob();
      }
    });
  }
}
