import {Usage} from './usage';
import {HostService} from './host-service';

export class Host {
    name: string = '';
    cpu: number = 0;
    mem: number = 0;
    usage: Usage = new Usage();
    services: HostService[] = [];
}
