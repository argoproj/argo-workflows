package hdfs

import (
	"fmt"

	hdfs "github.com/colinmarc/hdfs/v2"
	krb "github.com/jcmturner/gokrb5/v8/client"
	"github.com/jcmturner/gokrb5/v8/config"
)

func createHDFSClient(addresses []string, user string, dataTransferProtection string, krbOptions *KrbOptions) (*hdfs.Client, error) {
	options := hdfs.ClientOptions{
		Addresses:              addresses,
		DataTransferProtection: dataTransferProtection,
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
	krbConfig, err := config.NewFromString(krbOptions.Config)
	if err != nil {
		return nil, err
	}

	if krbOptions.CCacheOptions != nil {
		client, cacheErr := krb.NewFromCCache(&krbOptions.CCacheOptions.CCache, krbConfig)
		if cacheErr != nil {
			return nil, cacheErr
		}
		return client, nil
	} else if krbOptions.KeytabOptions != nil {
		client := krb.NewWithKeytab(
			krbOptions.KeytabOptions.Username,
			krbOptions.KeytabOptions.Realm,
			&krbOptions.KeytabOptions.Keytab,
			krbConfig,
		)
		err = client.Login()
		if err != nil {
			return nil, err
		}
		return client, nil
	}

	return nil, fmt.Errorf("failed to get a kerberos client")
}
