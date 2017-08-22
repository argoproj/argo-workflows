import { Component, Input, ElementRef, ViewChild, OnInit, OnDestroy, AfterViewInit, HostListener } from '@angular/core';
import { FormGroup, FormControl } from '@angular/forms';
import { URLSearchParams } from '@angular/http';

import { TaskService } from '../../services';
import { Deployment, Pod } from '../../model';

let Terminal = require('xterm');
Terminal.loadAddon('fit');

@Component({
    selector: 'ax-sys-console',
    templateUrl: './sys-console.html',
    styles: [ require('./sys-console.scss') ],
})
/**
 * Allows to start new process in job container using specified identifier. Uses web socket to send messages to the process and renders
 * processes stdout.
 */
export class SysConsoleComponent implements OnInit, OnDestroy, AfterViewInit {

    @ViewChild('container')
    public container: ElementRef;

    @Input()
    set setJobId(value: string) {
        if (value !== null) {
            this.jobId = value;
            if (this.terminal && this.jobId !== value) {
                this.ensureSocketClosed().then(() => {
                    this.terminal.clear();
                });
            }
        }
    }


    @Input()
    set consoleAccessSettings(val: { deployment: Deployment, pod: Pod }) {
        if (val) {
            this.settings = val;
            this.podObject = this.settings.pod;
            this.deploymentObject = this.settings.deployment;
            if (this.terminal) {
                this.ensureSocketClosed().then(() => {
                    this.terminal.clear();
                });
            }
        }
    }

    public jobId: string;
    public bashForm: FormGroup;

    private socket: WebSocket;
    private terminal: any;
    private settings: { deployment: Deployment, pod: Pod };
    private podObject: Pod;
    private deploymentObject: Deployment;

    constructor(private taskService: TaskService) { }

    public ngOnInit() {
        this.bashForm = new FormGroup({
            bashCommand: new FormControl('bash')
        });

        this.initTerminal();
    }

    public ngAfterViewInit() {
        this.terminal.fit();
    }

    public ngOnDestroy() {
        this.ensureSocketClosed();
        if (this.terminal) {
            this.terminal.destroy();
            this.terminal = null;
        }
    }

    get connected(): boolean {
        return !!this.socket;
    }

    @HostListener('window:resize', ['$event'])
    public onWindowResize() {
        if (this.terminal) {
            this.terminal.fit();
        }
    }

    public exec(form: FormGroup) {
        this.ensureSocketClosed().then(() => {
            this.terminal.clear();
            this.terminal.fit();
            this.terminal.writeln('Connecting...');
            this.socket = this.getSocket(form.value.bashCommand, this.termSize);
            this.bindSocketEvents();
        });
    }

    private initTerminal() {
        let size = this.termSize;
        this.terminal = new Terminal({
            cols: size.cols,
            rows: size.rows,
            screenKeys: true,
            useStyle: true,
            cursorBlink: true,
        });

        this.terminal.on('data', data => {
            if (this.socket) {
                this.socket.send(data);
            }
        });
        this.terminal.open(this.container.nativeElement);
    }

    private getSocket(command: string, size: { cols: number, rows: number }) {
        let search = new URLSearchParams();
        let url = '';
        search.set('cmd', command);
        search.set('h', size.rows.toFixed());
        search.set('w', size.cols.toFixed());

        if (this.jobId) {
            url = `v1/services/${this.jobId}/exec`;
        } else if (this.settings) {
            url = `v1/deployments/${this.deploymentObject.id}/exec`;
            search.set('instance', this.podObject.name);
            search.set('container', this.podObject.containers[0].name);
        }
        return this.taskService.connectToConsole(url, search);
    }

    private bindSocketEvents() {
        this.socket.onopen = () => {
            this.terminal.clear();
            this.terminal.fit();
            this.terminal.focus();
            this.socket.onmessage = evt => {
                if (evt.data instanceof ArrayBuffer) {
                    let bytearray = new Uint8Array(evt.data);
                    let result = '';
                    for (let i = 0; i < bytearray.length; i++) {
                        result += String.fromCharCode(bytearray[i]);
                    }
                    this.terminal.write(result);
                } else {
                    this.terminal.write(evt.data);
                }
                this.terminal.fit();
            };
            this.socket.onerror = evt => {
                this.terminal.writeln('Connection error');
            };
            this.socket.onclose = evt => {
                this.terminal.writeln('Session terminated');
                this.socket = null;
            };
        };
    }

    private get termSize(): { cols: number, rows: number } {
        return {
            cols: Math.round(this.container.nativeElement.clientWidth / 7),
            rows: Math.round(this.container.nativeElement.clientHeight / 20),
        };
    }

    private ensureSocketClosed(): Promise<any> {
        if (this.socket) {
            return new Promise<any>((resolve, reject) => {
                this.socket.onclose = () => {
                    resolve();
                };
                this.socket.close();
            });
        }
        return Promise.resolve();
    }
}
