import {EventEmitter, Component, Input} from '@angular/core';
import {ModalService} from '../../services';

@Component({
    selector: 'ax-user-add-modal',
    templateUrl: './modal-template.html'
})
export class ModalTemplateComponent {

    @Input() eventEmitter: EventEmitter<any>;
    private modal: Object = {};

    constructor(private modalService: ModalService) {
        this.modal = modalService.getModalConfig();
    }

    success() {
        this.modalService.resolve();
    }

    cancel() {
        this.modalService.reject();
    }
}
