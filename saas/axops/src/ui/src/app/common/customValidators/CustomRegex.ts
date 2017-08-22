export class CustomRegex {

    // Password will match only: 8+ letters, at least 1 lower case letter,
    // at least 1 upper case letter, and at least 1 special character
    public static password = /^(?=.*?[A-Za-z])(?=.*?[0-9])(?=.*?[#?!@()_+=;:"\'<>,./~{\[\\}|\]$%^&*-]).{8,}$/;
    // Minimum 8 characters at least 1 Alphabet, 1 Number and 1 Special Character:
    // public static password: string = '^(?=.*[A-Za-z])(?=.*\d)(?=.*[$@$!%*#?&])[A-Za-z\d$@$!%*#?&]{8,}$';

    // following regex matches the following:
    // http://www.example.com // https://www.example.com // http://example.com // http://www.example.com/kino
    // But will not accept:
    // www.example // http://www.example // http://examlpe
    public static url = /^(http|https|ftp):\/\/(([A-Z0-9][A-Z0-9_-]*)(\.[A-Z0-9][A-Z0-9_-]*)+)(:(\d+))?\/?/i;

    // Username allows words between 3 and 16 characters, lower and upper case, and '-' sign
    public static username: string = '[a-z0-9_-]{3,16}$';

    public static number: string = '^\d+$';

    public static nonNegativeIntegers: string = '[0-9]+';

    public static float: string = '^-?\d*(\.\d+)?$';

    public static trueOrFalse: string = '^(true|false)$';

    public static myToolsUsername: string = '.+';

    static passwordPattern = /^(?=.*[a-z])(?=.*[0-9])(?=.*[!@#\$%\^&\*])(?=.{8,})/;

    static emailPattern = new RegExp(''
        + /^(([^<>()\[\]\\.,;:\s@"]+(\.[^<>()\[\]\\.,;:\s@"]+)*)|(".+"))/.source
        + /@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$/.source);

    // RepoRegex allows words in between slashes, numbers,  and '-', '_' signs
    public static repo: string = '(\\/([\\w-_])+)+';

    // Email regex according to RFC 822
    public static email: string =
        '^[a-zA-Z0-9.!#$%&\'*+/=?^_`{|}~-]+@[a-zA-Z0-9]' +
        '(?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$';

    public static firstName: string = '[A-Z][a-zA-Z]*';

    public static lastName: string = '[a-zA-z]+([ \'-][a-zA-Z]+)*';

    public static fullName: string = '^[a-zA-Z ,.\'-]+$';

    public static templateName: string = '^([a-zA-Z0-9_ -])*';

    public static artifactTagName: string = '^([a-zA-Z0-9_-])*';

    public static policyName: string = '^([a-zA-Z0-9_ -])*';

    public static templateDescription: string = '^((?!%).)*$';

    public static templateInputName: string = '^[a-zA-Z0-9_-]*$';

    public static templateLinuxPath: string = '^(/[^/ ]*)+/?$';

    public static templateParameter: string = '%%[a-zA-Z0-9_]+%%$';

    public static templateVolumeName: string = '^[a-zA-Z0-9- ]*$';

    public static cidr = '(^$|\\d{1,3}(\\.\\d{1,3}){3}\\/\\d{1,2}$)';
}
