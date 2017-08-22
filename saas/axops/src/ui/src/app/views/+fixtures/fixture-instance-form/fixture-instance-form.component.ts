import { Component, Input } from '@angular/core';
import { FixtureInstance, FixtureClass, FixtureAttribute } from '../../../model';
import { FormGroup, FormControl, FormArray, Validators } from '@angular/forms';
import { CustomValidators } from 'ng2-validation';

@Component({
    selector: 'ax-fixture-instance-form',
    templateUrl: './fixture-instance-form.html',
    styles: [ require('./fixture-instance-form.scss') ]
})
export class FixtureInstanceFormComponent {

    public form: FormGroup;
    public attributes: { name: string, array: boolean, required: boolean, definition: FixtureAttribute }[] = [];
    public validated = false;

    private instance: FixtureInstance;

    @Input()
    public set fixtureInstance(val: FixtureInstance) {
        this.instance = val;
        this.refreshForm();
    }

    @Input()
    public set fixtureClass(val: FixtureClass) {
        this.attributes = Object.keys(val && val.attributes || {}).map(name => {
            let definition = val.attributes[name];
            let flags = (definition.flags || '').split(',');
            return {
                name: name,
                required: flags.indexOf('required') > -1,
                array: flags.indexOf('array') > -1,
                definition: val.attributes[name],
            };
        }).sort((first, second) => {
            return (second.required ? 1 : 0) - (first.required ? 1 : 0);
        });
        this.refreshForm();
    }

    public get valid() {
        this.validated = true;
        return this.form && this.form.valid;
    }

    public addArrayAttributeValue(name: string) {
        let control = <FormArray> this.form.controls[name];
        control.push(new FormControl(null, Validators.required));
    }

    public removeArrayAttributeValue(name: string, index: number) {
        let control = <FormArray> this.form.controls[name];
        control.removeAt(index);
    }

    public get value(): FixtureInstance {
        let attributes = Object.assign({}, this.form.value);
        delete attributes.name;
        delete attributes.description;
        delete attributes.concurrency;

        // make sure that bool attribute is bool
        this.attributes.filter(attr => attr.definition.type === 'bool').forEach(attr => {
            attributes[attr.name] = !!attributes[attr.name];
        });

        return {
            name: this.form.value.name,
            description: this.form.value.description,
            concurrency: this.form.value.concurrency,
            attributes,
        };
    }

    public reset() {
        this.refreshForm();
    }

    private refreshForm() {
        this.validated = false;
        this.form = new FormGroup({
            name: new FormControl(this.instance && this.instance.name || '', [ Validators.required ]),
            description: new FormControl(this.instance && this.instance.description || '', [ Validators.required ]),
            concurrency: new FormControl(this.instance && this.instance.concurrency || 1, [ Validators.required, CustomValidators.min(0) ]),
        });
        this.attributes.forEach(attribute => {
            let validators = [];
            if (attribute.required) {
                validators.push(Validators.required);
            }
            let instanceAttributes = this.instance && this.instance.attributes;
            if (attribute.array) {
                let value = instanceAttributes && instanceAttributes[attribute.name] || attribute.definition.default || [];
                this.form.addControl(attribute.name, new FormArray(value.map(item => new FormControl(item, [Validators.required]))));
            } else {
                let value = instanceAttributes && instanceAttributes[attribute.name] || attribute.definition.default || '';
                this.form.addControl(attribute.name, new FormControl(value, validators));
            }
        });
    }
}
