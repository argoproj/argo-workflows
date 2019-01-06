package hdfs

import (
	"fmt"

	"github.com/colinmarc/hdfs"
	krb "gopkg.in/jcmturner/gokrb5.v5/client"
	"gopkg.in/jcmturner/gokrb5.v5/config"
)

func createHDFSClient(addresses []string, user string, krbOptions *KrbOptions) (*hdfs.Client, error) {
	options := hdfs.ClientOptions{
		Addresses: addresses,
	}

	if krbOptions != nil {
		krbClient, err := createKrbClient(krbOptions)
		if err != nil {
			return nil, err
		}
		options.KerberosClient = krbClient
		options.KerberosServicePrincipleName = krbOptions.ServicePrincipalName
	} else {
		options.User = user
	}

	return hdfs.NewClient(options)
}

func createKrbClient(krbOptions *KrbOptions) (*krb.Client, error) {
	krbConfig, err := config.NewConfigFromString(krbOptions.Config)
	if err != nil {
		return nil, err
	}

	if krbOptions.CCacheOptions != nil {
		client, err := krb.NewClientFromCCache(krbOptions.CCacheOptions.CCache)
		if err != nil {
			return nil, err
		}
		return client.WithConfig(krbConfig), nil
	} else if krbOptions.KeytabOptions != nil {
		client := krb.NewClientWithKeytab(krbOptions.KeytabOptions.Username, krbOptions.KeytabOptions.Realm, krbOptions.KeytabOptions.Keytab)
		client = *client.WithConfig(krbConfig)
		err = client.Login()
		if err != nil {
			return nil, err
		}
		return &client, nil
	}

	return nil, fmt.Errorf("Failed to get a Kerberos client")
}
