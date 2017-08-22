export class Branch {
    public id: string;
    public repo: string;
    public name: string;
    public parent_id: string;
    public project: string;
    public roots: Branch[];
    public items: Branch[];
}
