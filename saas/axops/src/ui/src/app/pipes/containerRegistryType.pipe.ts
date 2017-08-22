import {Pipe, PipeTransform} from '@angular/core';
import {REGISTRY_TYPES} from '../model';

@Pipe({
    name: 'containerRegistryType'
})

export class ContainerRegistryTypePipe implements PipeTransform {
    transform(value: string, args: any[]) {
        let typeName = '';
        if (!value) {
            return '';
        }
        switch (value) {
            case REGISTRY_TYPES.dockerhub:
                typeName = 'Docker Hub';
                break;
            case REGISTRY_TYPES.privateRegistry:
                typeName = 'Private Registry';
                break;
        }

        return typeName;
    }
}
