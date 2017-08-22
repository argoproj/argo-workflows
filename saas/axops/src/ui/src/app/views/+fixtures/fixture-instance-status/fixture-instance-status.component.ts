import { Component, Input } from '@angular/core';
import { FixtureInstance, FixtureStatuses } from '../../../model';

@Component({
    selector: 'ax-fixture-instance-status',
    templateUrl: './fixture-instance-status.html',
    styles: [ require('./fixture-instance-status.scss') ]
})
export class FixtureInstanceStatusComponent {

    @Input()
    public fixture: FixtureInstance = new FixtureInstance();

    public get operationInProgress(): {name: string, id: string} {
        return this.fixture.status === FixtureStatuses.OPERATING && this.fixture.operation;
    }

    public get status(): string {
        let status: string;
        if (this.fixture.status === FixtureStatuses.ACTIVE) {
            let activeStatus = this.fixture.referrers.length > 0 ? 'In Use' : 'Available';
            status = `Active, ${ this.fixture.enabled ? activeStatus : 'Disabled' } `;
        } else {
            status = this.getStatusTitle(this.fixture.status);
            if (!this.fixture.enabled) {
                status = `${status}, Disabled`;
            }
        }
        return status;
    }

    public get fixtureStatusCode(): string {
        if (!this.fixture.enabled || this.fixture.status === FixtureStatuses.DELETED) {
            return 'disabled';
        } else if (this.fixture.status === FixtureStatuses.CREATE_ERROR || this.fixture.status === FixtureStatuses.DELETE_ERROR) {
            return 'error';
        } else if (this.fixture.status === FixtureStatuses.OPERATING) {
            return 'operating';
        } else {
            return 'active';
        }
    }

    private getStatusTitle(status: string) {
        switch (status) {
            case FixtureStatuses.INIT:
                return 'Initializing';
            case FixtureStatuses.CREATING:
                return 'Creating';
            case FixtureStatuses.CREATE_ERROR:
                return 'Fixture Creation Error';
            case FixtureStatuses.ACTIVE:
                return 'Active';
            case FixtureStatuses.OPERATING:
                return 'Operation In Progress';
            case FixtureStatuses.DELETING:
                return 'Deleting';
            case FixtureStatuses.DELETE_ERROR:
                return 'Fixture Deletion Error';
            case FixtureStatuses.DELETED:
                return 'Deleted';
        }
    }
}
