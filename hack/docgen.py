import json
import re
from pathlib import Path

SECTION_HEADER = """

# %s
"""

FIELD_HEADER = """

## %s

%s"""

FIELD_TABLE_HEADER = """

### Fields
| Field Name | Field Type | Description   |
|:----------:|:----------:|---------------|"""

TABLE_ROW = """
|`%s`|%s|%s|"""

DEP_TABLE_ROW = """
|~`%s`~|~%s~|%s|"""

DROPDOWN_OPENER = """
<details>
<summary>%s (click to open)</summary>
<br>"""

LIST_ELEMENT = """

- %s"""

DROPDOWN_CLOSER = """
</details>"""


def clean_title(title):
    if "\n+" in title:
        return title[:title.find("\n+")]
    return title


def clean_desc(desc):
    desc = desc.replace("\n", "")
    dep = ""
    if 'DEPRECATED' in desc:
        dep = " " + desc[desc.find("DEPRECATED"):]

    if '+patch' in desc:
        desc = desc[:desc.find("+patch")]
    if '+proto' in desc:
        desc = desc[:desc.find("+proto")]
    if '+option' in desc:
        desc = desc[:desc.find("+option")]

    if dep != "" and 'DEPRECATED' not in desc:
        desc += dep
    return desc


def get_row(name, _type, desc):
    if 'DEPRECATED' in desc:
        dep = desc.find('DEPRECATED')
        return DEP_TABLE_ROW % (name, _type, "~" + desc[:dep] + "~ " + desc[dep:])
    else:
        return TABLE_ROW % (name, _type, desc)


def get_name_from_full_name(ref):
    return ref.split('.')[-1]


def get_desc(key, defs):
    if 'description' in defs[key]:
        return clean_desc(defs[key]['description'])
    elif 'title' in defs[key]:
        return clean_desc(clean_title(defs[key]['title']))
    return "_No description available_"


def link(text, link_to):
    return "[%s](%s)" % (text, link_to)


def get_desc_from_field(field):
    if 'description' in field:
        return field['description']
    elif 'title' in field:
        return field['title']
    return "_No desription available_"


def get_examples(examples, summary):
    out = DROPDOWN_OPENER % summary
    for example in examples:
        name = example.split("/")[-1]
        out += LIST_ELEMENT % link("`%s`" % name, "../" + example)
    out += DROPDOWN_CLOSER
    return out


def get_key_value_field_types(field):
    key_type, val_type = "string", "string"
    if 'type' in field['additionalProperties']:
        key_type = field['additionalProperties']['type']
    if 'format' in field['additionalProperties']:
        val_type = field['additionalProperties']['format']
    return key_type, val_type


def get_object_type(field, json_field_name, add_to_queue):
    obj_type_raw = field['type']
    if obj_type_raw == "array":
        if '$ref' in field['items']:
            ref = field['items']['$ref'][14:]
            add_to_queue(ref, json_field_name)

            if ref == "io.argoproj.workflow.v1alpha1.WorkflowStep":
                return "`Array<Array<`%s`>>`" % link("`%s`" % get_name_from_full_name(ref),
                                                     "#" + get_name_from_full_name(ref).lower())

            return "`Array<`%s`>`" % link("`%s`" % get_name_from_full_name(ref),
                                          "#" + get_name_from_full_name(ref).lower())

        return "`Array< %s >`" % get_name_from_full_name(field['items']['type'])

    elif obj_type_raw == "object":
        if '$ref' in field['additionalProperties']:
            ref = field['additionalProperties']['$ref'][14:]
            add_to_queue(ref, json_field_name)
            return link("`%s`" % get_name_from_full_name(ref), "#" + get_name_from_full_name(ref).lower())

        else:
            return "`Map< %s , %s >`" % get_key_value_field_types(field)

    elif 'format' in field:
        return "`%s`" % field['format']

    return "`%s`" % field['type']


class DocGeneratorContext:

    def __init__(self):
        self.defs = {}
        self.completed_fields = set()
        self.json_name = {}
        self.queue = ['io.argoproj.workflow.v1alpha1.Workflow', 'io.argoproj.workflow.v1alpha1.CronWorkflow',
                      'io.argoproj.workflow.v1alpha1.WorkflowTemplate']
        self.external = []
        self.index = {}

    def run(self):
        self.load_files()

        out = SECTION_HEADER % "Argo Fields"
        while self.queue:
            out += self.get_template(self.queue.pop(0))

        out += SECTION_HEADER % "External Fields"
        while self.external:
            out += self.get_template(self.external.pop(0))

        return out

    def load_files(self):
        with open('api/openapi-spec/swagger.json') as swagger:
            s = json.load(swagger)
            self.defs = s['definitions']

        for file_name_poisx in Path('examples').rglob('*.yaml'):
            file_name = str(file_name_poisx)
            with open(file_name) as wf:
                wf_text = wf.read()
                kinds = re.findall(r"kind: ([a-zA-Z]+)", wf_text)
                for kind in kinds:
                    if kind not in self.index:
                        self.index[kind] = set()
                    self.index[kind].add(file_name)

                finds = re.findall(r"([a-zA-Z]+?):", wf_text)
                for find in finds:
                    if find not in self.index:
                        self.index[find] = set()
                    self.index[find].add(file_name)

    def add_to_queue(self, ref, json_field_name):
        if ref == "io.argoproj.workflow.v1alpha1.ParallelSteps":
            ref = "io.argoproj.workflow.v1alpha1.WorkflowStep"
        if ref not in self.completed_fields:
            self.completed_fields.add(ref)
            self.json_name[ref] = json_field_name
            if "io.argoproj.workflow" in ref:
                self.queue.append(ref)
            else:
                self.external.append(ref)

    def get_template(self, key):
        out = FIELD_HEADER % (get_name_from_full_name(key), get_desc(key, self.defs))

        if get_name_from_full_name(key) in self.index:
            out += get_examples(self.index[get_name_from_full_name(key)], "Examples")

        if key in self.json_name and self.json_name[key] in self.index:
            out += get_examples(self.index[self.json_name[key]], "Examples with this field")

        if 'properties' not in self.defs[key]:
            return out

        out += FIELD_TABLE_HEADER
        for json_field_name, field in self.defs[key]['properties'].items():
            if '$ref' in field:

                ref = field['$ref'][14:]
                self.add_to_queue(ref, json_field_name)

                desc = get_desc_from_field(field)
                out += get_row(json_field_name,
                               link("`%s`" % get_name_from_full_name(ref), "#" + get_name_from_full_name(ref).lower()),
                               clean_desc(desc))

            else:
                obj_type = get_object_type(field, json_field_name, lambda r, j: self.add_to_queue(r, j))
                desc = get_desc_from_field(field)

                out += get_row(json_field_name, obj_type, clean_desc(desc))

        return out

# This file should be run with `make docs` on the root directory

ctx = DocGeneratorContext()
fields = ctx.run()
with open('docs/fields.md', 'w') as file:
    file.write(fields)
