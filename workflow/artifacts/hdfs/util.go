package hdfs

import (
	"fmt"

	hdfsv2 "github.com/colinmarc/hdfs/v2"
	krbv8 "github.com/jcmturner/gokrb5/v8/client"
	configv8 "github.com/jcmturner/gokrb5/v8/config"
)

func createHDFSClient(addresses []string, user string, krbOptions *KrbOptions) (*hdfsv2.Client, error) {
	options := hdfsv2.ClientOptions{
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

	return hdfsv2.NewClient(options)
}

func createKrbClient(krbOptions *KrbOptions) (*krbv8.Client, error) {
	krbConfig, err := configv8.NewFromString(krbOptions.Config)
	if err != nil {
		return nil, err
	}

	if krbOptions.CCacheOptions != nil {
		client, err := krbv8.NewFromCCache(krbOptions.CCacheOptions.CCache, krbConfig)
		if err != nil {
			return nil, err
		}
		return client, nil
	}
	if krbOptions.KeytabOptions != nil {
		client := krbv8.NewWithKeytab(krbOptions.KeytabOptions.Username, krbOptions.KeytabOptions.Realm, krbOptions.KeytabOptions.Keytab, krbConfig)
		err = client.Login()
		if err != nil {
			return nil, err
		}
		return client, nil
	}

	return nil, fmt.Errorf("failed to get a Kerberos client")
}
