import { Component, Input, OnInit, Output, EventEmitter } from '@angular/core';
import { FormGroup, FormControl, Validators } from '@angular/forms';

import { Group } from '../../../model';
import { GroupService, NotificationService, UsersService } from '../../../services';
import { CustomRegex } from '../../../common/customValidators/CustomRegex';

@Component({
    selector: 'ax-invite-panel',
    templateUrl: './invite.html',
    styles: [ require('./invite.scss') ],
})
export class InviteUserComponent implements OnInit {

    @Input()
    public show: boolean;

    @Output()
    public onClose: EventEmitter<null> = new EventEmitter();

    public submitted: boolean;
    private inviteForm: FormGroup;
    private userGroups: string[] = [];

    constructor(private usersService: UsersService,
                private groupService: GroupService,
                private notificationService: NotificationService) {
        this.inviteForm = new FormGroup({
            username: new FormControl('', [ Validators.required, Validators.pattern(CustomRegex.email) ]),
            group: new FormControl('', Validators.required),
            isMailingGroup: new FormControl(false),
            firstName: new FormControl(''),
            lastName: new FormControl(''),
        });
    }

    ngOnInit() {
        this.getGroups();
    }

    getGroups() {
        this.groupService.getGroups().subscribe(result => {
            this.userGroups = Group.getGroupList(result.data);
        });
    }

    inviteUser(form: FormGroup) {
        this.submitted = true;
        if (form.valid) {
            let isMailingGroup = form.value.isMailingGroup;
            let firstName = isMailingGroup ? null : form.value.firstName;
            let lastName = isMailingGroup ? null : form.value.lastName;
            this.usersService.inviteUser(form.value.username, form.value.group, !isMailingGroup, firstName, lastName)
                .subscribe(
                    success => {
                        this.notificationService.showNotification.emit(
                            {
                                message: `User ${form.value.username} has been invited.`,
                            });
                        this.close();
                    },
                );
        }
    }

    close() {
        this.onClose.emit();
    }
}
