import { FormControl, AbstractControl, ValidatorFn, Validators } from '@angular/forms';
import { CustomRegex } from './CustomRegex';

export class CustomValidators {

    public static validateListOf(objectType: string): ValidatorFn {
        return (control: AbstractControl): {[key: string]: any} => {
            // TODO improve if statement
            if (control.value.toString() !== 'true' // value can't be 'true' and 'false'
                && control.value.toString() !== 'false'
                && control.value.trim()[0] !== '"' // value can't be in a doublequotes
                && CustomValidators.isJson(control.value)) {
                let convertedToList = JSON.parse(control.value);
                let itemsCorrectionList: boolean[] = [];
                convertedToList.forEach(item => {
                    itemsCorrectionList.push((typeof item).toString() === objectType);
                });

                return itemsCorrectionList.indexOf(false) !== -1 ? { error: true, expected_type: objectType } : null;
            } else {
                return { isNotArrayOfSetType: { error: true, expected_type: objectType } };
            }
        };
    }

    public static number(prms: {min?: number, max?: number}): ValidatorFn {
        return (control: AbstractControl): {[key: string]: any} => {
            let val: number = control.value;
            val = val || 0;

            if (isNaN(val) || /\D/.test(val.toString())) {
                return { number: true };
            } else if (!isNaN(prms.min) && !isNaN(prms.max)) {
                return val < prms.min || val > prms.max ? {number: true} : null;
            } else if (!isNaN(prms.min)) {
                return val < prms.min ? {number: true} : null;
            } else if (!isNaN(prms.max)) {
                return val > prms.max ? {number: true} : null;
            } else {
                return null;
            }
        };
    }

    public static requiredIf(condition: boolean): ValidatorFn {
        return (control: AbstractControl): {[key: string]: any} => {
             if (condition) {
                 return Validators.required(control);
             }
        };
    }


    public static validateType(objectType: string): ValidatorFn {
        return (control: AbstractControl): {[key: string]: any} => {
            if (CustomValidators.isJson(control.value)) {
                return typeof JSON.parse(control.value) === objectType ? null : { wrongType: { error: true, expected_type: objectType } };
            }
            return control.value ? { wrongType: { error: true, expected_type: objectType } } : null;
        };
    }

    public static isJson(str) {
        try {
            JSON.parse(str);
        } catch (e) {
            return false;
        }
        return true;
    }

    static emailValidator(control: FormControl): {[key: string]: any} {
        if (control.value && !CustomRegex.emailPattern.test(control.value)) {
            return {invalidEmail: true};
        }
    }

    static matchingPasswords(group: AbstractControl): {[key: string]: any} {
        if (group.value.password !== group.value.repeat_password) {
            return {
                mismatchedPasswords: true
            };
        }
    }

    public static matchProperties(propertyFirst: string, propertySecond: string): ValidatorFn {
        return (group: AbstractControl): {[key: string]: any} => {
            if (group.value[propertyFirst] !== group.value[propertySecond]) {
                return {
                    mismatchedProperties: true, firstComparedProperty: propertyFirst, secondComparedProperty: propertySecond
                };
            } else {
                return null;
            }
        };
    }
}
