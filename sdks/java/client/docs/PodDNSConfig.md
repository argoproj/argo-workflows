

# PodDNSConfig

PodDNSConfig defines the DNS parameters of a pod in addition to those generated from DNSPolicy.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**nameservers** | **List&lt;String&gt;** | A list of DNS name server IP addresses. This will be appended to the base nameservers generated from DNSPolicy. Duplicated nameservers will be removed. |  [optional]
**options** | [**List&lt;PodDNSConfigOption&gt;**](PodDNSConfigOption.md) | A list of DNS resolver options. This will be merged with the base options generated from DNSPolicy. Duplicated entries will be removed. Resolution options given in Options will override those that appear in the base DNSPolicy. |  [optional]
**searches** | **List&lt;String&gt;** | A list of DNS search domains for host-name lookup. This will be appended to the base search paths generated from DNSPolicy. Duplicated search paths will be removed. |  [optional]



