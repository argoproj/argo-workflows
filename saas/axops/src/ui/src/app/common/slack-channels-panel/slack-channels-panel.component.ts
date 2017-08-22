import { Component, Input, Output, EventEmitter } from '@angular/core';

import { SlackService } from '../../services';

@Component({
    selector: 'ax-slack-channels-panel',
    templateUrl: './slack-channels-panel.html',
})
export class SlackChannelsPanelComponent {
    @Input()
    public selectedChannels: string[];

    @Input()
    set show(val: boolean) {
        this.isPanelVisible = val;
        if (!this.channels.length && this.isPanelVisible) {
            this.getChannels();
        } else if (this.channels.length) {
            this.channelsToDisplay = JSON.parse(JSON.stringify(this.channels));
            this.selectChannels();
        }
    }

    @Output()
    public onChange: EventEmitter<string[]> = new EventEmitter();

    @Output()
    public onClose: EventEmitter<any> = new EventEmitter();

    public isPanelVisible: boolean = false;
    public searchedUser: string;
    public getChannelsLoader: boolean = false;
    private channels: { name: string, checked: boolean, display?: boolean }[] = [];
    private channelsToDisplay: { name: string, checked: boolean, display?: boolean }[] = [];

    constructor(private slackService: SlackService) {}

    public add() {
        this.onChange.emit(this.channelsToDisplay.filter(channel => channel.checked).map(channel => channel.name));
        this.closeSlackChannelsSlidingPanel();
    }

    public closeSlackChannelsSlidingPanel() {
        this.onClose.emit();
    }

    public changed(searchString: string) {
        this.channelsToDisplay.forEach(channel => { channel.display = channel.name.toLowerCase().indexOf(searchString.toLowerCase()) !== -1; });
    }

    private async getChannels() {
        this.getChannelsLoader = true;
        let channels = await this.slackService.getSlackChannels();
        this.channels = channels.map(channel => {
            return { name: channel, checked: false, display: true };
        });
        this.channelsToDisplay = JSON.parse(JSON.stringify(this.channels));
        this.selectChannels();
        this.getChannelsLoader = false;
    }

    private selectChannels() {
        let channels = this.selectedChannels.map(channel => channel.substring(0, channel.indexOf('@slack')));
        this.channelsToDisplay.forEach(channel => {
            channel.checked = channels.indexOf(channel.name) !== -1;
        });
    }
}
