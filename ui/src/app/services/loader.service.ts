import { Injectable } from '@angular/core';
import { setTimeout } from 'timers';

let backgroudJobsCount = 0;

@Injectable()
export class LoaderService {

  public get isLoaderVisible() {
    return backgroudJobsCount > 0;
  }

  public startBackgroudJob() {
    setTimeout(() => backgroudJobsCount++, 0);
  }

  public completeBackgroudJob() {
    setTimeout(() => backgroudJobsCount--, 0);
  }
}
