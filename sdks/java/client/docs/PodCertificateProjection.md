

# PodCertificateProjection

PodCertificateProjection provides a private key and X.509 certificate in the pod filesystem.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**certificateChainPath** | **String** | Write the certificate chain at this path in the projected volume.  Most applications should use credentialBundlePath.  When using keyPath and certificateChainPath, your application needs to check that the key and leaf certificate are consistent, because it is possible to read the files mid-rotation. |  [optional]
**credentialBundlePath** | **String** | Write the credential bundle at this path in the projected volume.  The credential bundle is a single file that contains multiple PEM blocks. The first PEM block is a PRIVATE KEY block, containing a PKCS#8 private key.  The remaining blocks are CERTIFICATE blocks, containing the issued certificate chain from the signer (leaf and any intermediates).  Using credentialBundlePath lets your Pod&#39;s application code make a single atomic read that retrieves a consistent key and certificate chain.  If you project them to separate files, your application code will need to additionally check that the leaf certificate was issued to the key. |  [optional]
**keyPath** | **String** | Write the key at this path in the projected volume.  Most applications should use credentialBundlePath.  When using keyPath and certificateChainPath, your application needs to check that the key and leaf certificate are consistent, because it is possible to read the files mid-rotation. |  [optional]
**keyType** | **String** | The type of keypair Kubelet will generate for the pod.  Valid values are \&quot;RSA3072\&quot;, \&quot;RSA4096\&quot;, \&quot;ECDSAP256\&quot;, \&quot;ECDSAP384\&quot;, \&quot;ECDSAP521\&quot;, and \&quot;ED25519\&quot;. | 
**maxExpirationSeconds** | **Integer** | maxExpirationSeconds is the maximum lifetime permitted for the certificate.  Kubelet copies this value verbatim into the PodCertificateRequests it generates for this projection.  If omitted, kube-apiserver will set it to 86400(24 hours). kube-apiserver will reject values shorter than 3600 (1 hour).  The maximum allowable value is 7862400 (91 days).  The signer implementation is then free to issue a certificate with any lifetime *shorter* than MaxExpirationSeconds, but no shorter than 3600 seconds (1 hour).  This constraint is enforced by kube-apiserver. &#x60;kubernetes.io&#x60; signers will never issue certificates with a lifetime longer than 24 hours. |  [optional]
**signerName** | **String** | Kubelet&#39;s generated CSRs will be addressed to this signer. | 
**userAnnotations** | **Map&lt;String, String&gt;** | userAnnotations allow pod authors to pass additional information to the signer implementation.  Kubernetes does not restrict or validate this metadata in any way.  These values are copied verbatim into the &#x60;spec.unverifiedUserAnnotations&#x60; field of the PodCertificateRequest objects that Kubelet creates.  Entries are subject to the same validation as object metadata annotations, with the addition that all keys must be domain-prefixed. No restrictions are placed on values, except an overall size limitation on the entire field.  Signers should document the keys and values they support. Signers should deny requests that contain keys they do not recognize. |  [optional]



