package fixtures

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

type record struct {
	fields  log.Fields
	message string
}

type Diagnostics struct {
	records []record
}

func (d *Diagnostics) Logf(format string, args ...interface{}) {
	for _, line := range strings.Split(fmt.Sprintf(format, args...), "\n") {
		d.Log(log.Fields{}, line)
	}
}

func (d *Diagnostics) Log(context log.Fields, message string) {
	d.records = append(d.records, record{fields: context, message: message})
}

func (d *Diagnostics) Print() {
	for _, r := range d.records {
		log.WithFields(r.fields).Info(r.message)
	}
}
