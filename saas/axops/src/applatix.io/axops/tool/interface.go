// Copyright 2015-2016 Applatix, Inc. All rights reserved.
package tool

import "applatix.io/axerror"

type Tool interface {
	GetID() string
	GetURL() string
	GetCategory() string
	GetType() string
	GetPassword() string
	// Generat UUID, used by the Create method in ToolBase
	GenerateUUID()
	//Create(Tool) (*axerror.AXError, int)
	//Update(Tool) (*axerror.AXError, int)
	//Delete() (*axerror.AXError, int)

	// omit the sensitive information, eg. config secret key, password
	Omit()
	// test the connection or correctness, eg. test to see if the github credential is valid
	Test() (*axerror.AXError, int)

	//// persist the config
	//save() (*axerror.AXError, int)

	// pre-process the configuration, eg. fetch the repository list for SCM configs
	pre() (*axerror.AXError, int)
	// validate attributes, eg. type, category are required
	validate() (*axerror.AXError, int)
	// push the updated configuration to the 3rd party, eg. axnotification needs the SMTP config
	PushUpdate() (*axerror.AXError, int)
	// notify the configuration deletion to the 3rd part
	pushDelete() (*axerror.AXError, int)
	// post-process the configuration, eg. disable commits after repo config is deleted
	Post(old, new interface{}) (*axerror.AXError, int)
}
