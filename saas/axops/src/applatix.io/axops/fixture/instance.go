// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package fixture

type Instance struct {
	ID            string    `json:"id,omitempty"`
	Name          string    `json:"name,omitempty"`
	Description   string    `json:"description,omitempty"`
	ClassID       string    `json:"class_id,omitempty"`
	ClassName     string    `json:"class_name,omitempty"`
	Enabled       *bool     `json:"enabled,omitempty"`
	DisableReason string    `json:"disable_reason,omitempty"`
	Owner         string    `json:"owner,omitempty"`
	Creator       string    `json:"creator,omitempty"`
	Status        string    `json:"status,omitempty"`
	StatusDetail  TypeMap   `json:"status_detail,omitempty"`
	Concurrency   int       `json:"concurrency,omitempty"`
	Referrers     []TypeMap `json:"referrers,omitempty"`
	Operation     TypeMap   `json:"operation,omitempty"`
	Attributes    TypeMap   `json:"attributes,omitempty"`
	Ctime         int64     `json:"ctime,omitempty"`
	Mtime         int64     `json:"mtime,omitempty"`
	Atime         int64     `json:"atime,omitempty"`
}
