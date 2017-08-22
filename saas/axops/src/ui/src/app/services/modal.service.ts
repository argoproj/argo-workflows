import { EventEmitter, Inject } from '@angular/core';
import { EventsService } from './events.service';


export class ModalService {
    private modalStream: EventEmitter<any>;
    private modalConfig: Object;
    constructor( @Inject(EventsService) private eventsService: EventsService) {
    }

    showModal(title, header, message = '', icon = { name: null, color: null }, okButton?: boolean) {
        this.modalConfig = {
            title: title,
            header: header,
            message: message,
            icon: icon,
        };

        if (okButton) {
            this.modalConfig['okButton'] = okButton;
        }

        this.modalStream = new EventEmitter();
        this.eventsService.modal.emit(
            this.modalStream
        );

        return this.modalStream;
    }

    copyModal(title, header, message = '', icon = { name: null, color: null }, okButton?: boolean) {
        this.modalConfig = {
            copy: true,
            title: title,
            header: header,
            message: message,
            icon: icon,
        };

        if (okButton) {
            this.modalConfig['okButton'] = okButton;
        }

        this.modalStream = new EventEmitter();
        this.eventsService.modal.emit(
            this.modalStream
        );

        return this.modalStream;
    }

    resolve() {
        this.modalStream.emit(true);
    }

    reject() {
        this.modalStream.emit(false);
    }

    getModalConfig() {
        return this.modalConfig;
    }
}
