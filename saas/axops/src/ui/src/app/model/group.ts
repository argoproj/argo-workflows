import { USER_GROUPS } from '.';

export class Group {
    omitempty: string[] = [];
    id: string = '';
    name: string = '';

    public static getGroupList(groups: Group[]): string[] {
        return groups.filter(group => group.name !== USER_GROUPS.super_admin).map(group => {
            return group.name;
        });
    }
}
