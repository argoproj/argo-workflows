// +build !ignore_autogenerated

// Code generated by deepcopy-gen. DO NOT EDIT.

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ArchiveStrategy) DeepCopyInto(out *ArchiveStrategy) {
	*out = *in
	if in.Tar != nil {
		in, out := &in.Tar, &out.Tar
		*out = new(TarStrategy)
		**out = **in
	}
	if in.None != nil {
		in, out := &in.None, &out.None
		*out = new(NoneStrategy)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ArchiveStrategy.
func (in *ArchiveStrategy) DeepCopy() *ArchiveStrategy {
	if in == nil {
		return nil
	}
	out := new(ArchiveStrategy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Arguments) DeepCopyInto(out *Arguments) {
	*out = *in
	if in.Parameters != nil {
		in, out := &in.Parameters, &out.Parameters
		*out = make([]Parameter, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Artifacts != nil {
		in, out := &in.Artifacts, &out.Artifacts
		*out = make([]Artifact, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Arguments.
func (in *Arguments) DeepCopy() *Arguments {
	if in == nil {
		return nil
	}
	out := new(Arguments)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Artifact) DeepCopyInto(out *Artifact) {
	*out = *in
	if in.Mode != nil {
		in, out := &in.Mode, &out.Mode
		*out = new(int32)
		**out = **in
	}
	in.ArtifactLocation.DeepCopyInto(&out.ArtifactLocation)
	if in.Archive != nil {
		in, out := &in.Archive, &out.Archive
		*out = new(ArchiveStrategy)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Artifact.
func (in *Artifact) DeepCopy() *Artifact {
	if in == nil {
		return nil
	}
	out := new(Artifact)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ArtifactLocation) DeepCopyInto(out *ArtifactLocation) {
	*out = *in
	if in.ArchiveLogs != nil {
		in, out := &in.ArchiveLogs, &out.ArchiveLogs
		*out = new(bool)
		**out = **in
	}
	if in.S3 != nil {
		in, out := &in.S3, &out.S3
		*out = new(S3Artifact)
		(*in).DeepCopyInto(*out)
	}
	if in.Git != nil {
		in, out := &in.Git, &out.Git
		*out = new(GitArtifact)
		(*in).DeepCopyInto(*out)
	}
	if in.HTTP != nil {
		in, out := &in.HTTP, &out.HTTP
		*out = new(HTTPArtifact)
		**out = **in
	}
	if in.Artifactory != nil {
		in, out := &in.Artifactory, &out.Artifactory
		*out = new(ArtifactoryArtifact)
		(*in).DeepCopyInto(*out)
	}
	if in.HDFS != nil {
		in, out := &in.HDFS, &out.HDFS
		*out = new(HDFSArtifact)
		(*in).DeepCopyInto(*out)
	}
	if in.AzureBlob != nil {
		in, out := &in.AzureBlob, &out.AzureBlob
		*out = new(AzureBlobArtifact)
		(*in).DeepCopyInto(*out)
	}
	if in.Raw != nil {
		in, out := &in.Raw, &out.Raw
		*out = new(RawArtifact)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ArtifactLocation.
func (in *ArtifactLocation) DeepCopy() *ArtifactLocation {
	if in == nil {
		return nil
	}
	out := new(ArtifactLocation)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ArtifactRepositoryRef) DeepCopyInto(out *ArtifactRepositoryRef) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ArtifactRepositoryRef.
func (in *ArtifactRepositoryRef) DeepCopy() *ArtifactRepositoryRef {
	if in == nil {
		return nil
	}
	out := new(ArtifactRepositoryRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ArtifactoryArtifact) DeepCopyInto(out *ArtifactoryArtifact) {
	*out = *in
	in.ArtifactoryAuth.DeepCopyInto(&out.ArtifactoryAuth)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ArtifactoryArtifact.
func (in *ArtifactoryArtifact) DeepCopy() *ArtifactoryArtifact {
	if in == nil {
		return nil
	}
	out := new(ArtifactoryArtifact)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ArtifactoryAuth) DeepCopyInto(out *ArtifactoryAuth) {
	*out = *in
	if in.UsernameSecret != nil {
		in, out := &in.UsernameSecret, &out.UsernameSecret
		*out = new(v1.SecretKeySelector)
		(*in).DeepCopyInto(*out)
	}
	if in.PasswordSecret != nil {
		in, out := &in.PasswordSecret, &out.PasswordSecret
		*out = new(v1.SecretKeySelector)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ArtifactoryAuth.
func (in *ArtifactoryAuth) DeepCopy() *ArtifactoryAuth {
	if in == nil {
		return nil
	}
	out := new(ArtifactoryAuth)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AzureBlobArtifact) DeepCopyInto(out *AzureBlobArtifact) {
	*out = *in
	in.AccountNameSecret.DeepCopyInto(&out.AccountNameSecret)
	in.AccountKeySecret.DeepCopyInto(&out.AccountKeySecret)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AzureBlobArtifact.
func (in *AzureBlobArtifact) DeepCopy() *AzureBlobArtifact {
	if in == nil {
		return nil
	}
	out := new(AzureBlobArtifact)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ContinueOn) DeepCopyInto(out *ContinueOn) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ContinueOn.
func (in *ContinueOn) DeepCopy() *ContinueOn {
	if in == nil {
		return nil
	}
	out := new(ContinueOn)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DAGTask) DeepCopyInto(out *DAGTask) {
	*out = *in
	in.Arguments.DeepCopyInto(&out.Arguments)
	if in.TemplateRef != nil {
		in, out := &in.TemplateRef, &out.TemplateRef
		*out = new(TemplateRef)
		**out = **in
	}
	if in.Dependencies != nil {
		in, out := &in.Dependencies, &out.Dependencies
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.WithItems != nil {
		in, out := &in.WithItems, &out.WithItems
		*out = make([]Item, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.WithSequence != nil {
		in, out := &in.WithSequence, &out.WithSequence
		*out = new(Sequence)
		**out = **in
	}
	if in.ContinueOn != nil {
		in, out := &in.ContinueOn, &out.ContinueOn
		*out = new(ContinueOn)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DAGTask.
func (in *DAGTask) DeepCopy() *DAGTask {
	if in == nil {
		return nil
	}
	out := new(DAGTask)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DAGTemplate) DeepCopyInto(out *DAGTemplate) {
	*out = *in
	if in.Tasks != nil {
		in, out := &in.Tasks, &out.Tasks
		*out = make([]DAGTask, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.FailFast != nil {
		in, out := &in.FailFast, &out.FailFast
		*out = new(bool)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DAGTemplate.
func (in *DAGTemplate) DeepCopy() *DAGTemplate {
	if in == nil {
		return nil
	}
	out := new(DAGTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GitArtifact) DeepCopyInto(out *GitArtifact) {
	*out = *in
	if in.Depth != nil {
		in, out := &in.Depth, &out.Depth
		*out = new(uint)
		**out = **in
	}
	if in.Fetch != nil {
		in, out := &in.Fetch, &out.Fetch
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.UsernameSecret != nil {
		in, out := &in.UsernameSecret, &out.UsernameSecret
		*out = new(v1.SecretKeySelector)
		(*in).DeepCopyInto(*out)
	}
	if in.PasswordSecret != nil {
		in, out := &in.PasswordSecret, &out.PasswordSecret
		*out = new(v1.SecretKeySelector)
		(*in).DeepCopyInto(*out)
	}
	if in.SSHPrivateKeySecret != nil {
		in, out := &in.SSHPrivateKeySecret, &out.SSHPrivateKeySecret
		*out = new(v1.SecretKeySelector)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GitArtifact.
func (in *GitArtifact) DeepCopy() *GitArtifact {
	if in == nil {
		return nil
	}
	out := new(GitArtifact)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HDFSArtifact) DeepCopyInto(out *HDFSArtifact) {
	*out = *in
	in.HDFSConfig.DeepCopyInto(&out.HDFSConfig)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HDFSArtifact.
func (in *HDFSArtifact) DeepCopy() *HDFSArtifact {
	if in == nil {
		return nil
	}
	out := new(HDFSArtifact)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HDFSConfig) DeepCopyInto(out *HDFSConfig) {
	*out = *in
	in.HDFSKrbConfig.DeepCopyInto(&out.HDFSKrbConfig)
	if in.Addresses != nil {
		in, out := &in.Addresses, &out.Addresses
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HDFSConfig.
func (in *HDFSConfig) DeepCopy() *HDFSConfig {
	if in == nil {
		return nil
	}
	out := new(HDFSConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HDFSKrbConfig) DeepCopyInto(out *HDFSKrbConfig) {
	*out = *in
	if in.KrbCCacheSecret != nil {
		in, out := &in.KrbCCacheSecret, &out.KrbCCacheSecret
		*out = new(v1.SecretKeySelector)
		(*in).DeepCopyInto(*out)
	}
	if in.KrbKeytabSecret != nil {
		in, out := &in.KrbKeytabSecret, &out.KrbKeytabSecret
		*out = new(v1.SecretKeySelector)
		(*in).DeepCopyInto(*out)
	}
	if in.KrbConfigConfigMap != nil {
		in, out := &in.KrbConfigConfigMap, &out.KrbConfigConfigMap
		*out = new(v1.ConfigMapKeySelector)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HDFSKrbConfig.
func (in *HDFSKrbConfig) DeepCopy() *HDFSKrbConfig {
	if in == nil {
		return nil
	}
	out := new(HDFSKrbConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HTTPArtifact) DeepCopyInto(out *HTTPArtifact) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HTTPArtifact.
func (in *HTTPArtifact) DeepCopy() *HTTPArtifact {
	if in == nil {
		return nil
	}
	out := new(HTTPArtifact)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Inputs) DeepCopyInto(out *Inputs) {
	*out = *in
	if in.Parameters != nil {
		in, out := &in.Parameters, &out.Parameters
		*out = make([]Parameter, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Artifacts != nil {
		in, out := &in.Artifacts, &out.Artifacts
		*out = make([]Artifact, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Inputs.
func (in *Inputs) DeepCopy() *Inputs {
	if in == nil {
		return nil
	}
	out := new(Inputs)
	in.DeepCopyInto(out)
	return out
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Item.
func (in *Item) DeepCopy() *Item {
	if in == nil {
		return nil
	}
	out := new(Item)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Metadata) DeepCopyInto(out *Metadata) {
	*out = *in
	if in.Annotations != nil {
		in, out := &in.Annotations, &out.Annotations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Labels != nil {
		in, out := &in.Labels, &out.Labels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Metadata.
func (in *Metadata) DeepCopy() *Metadata {
	if in == nil {
		return nil
	}
	out := new(Metadata)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeStatus) DeepCopyInto(out *NodeStatus) {
	*out = *in
	if in.TemplateRef != nil {
		in, out := &in.TemplateRef, &out.TemplateRef
		*out = new(TemplateRef)
		**out = **in
	}
	in.StartedAt.DeepCopyInto(&out.StartedAt)
	in.FinishedAt.DeepCopyInto(&out.FinishedAt)
	if in.Daemoned != nil {
		in, out := &in.Daemoned, &out.Daemoned
		*out = new(bool)
		**out = **in
	}
	if in.Inputs != nil {
		in, out := &in.Inputs, &out.Inputs
		*out = new(Inputs)
		(*in).DeepCopyInto(*out)
	}
	if in.Outputs != nil {
		in, out := &in.Outputs, &out.Outputs
		*out = new(Outputs)
		(*in).DeepCopyInto(*out)
	}
	if in.Children != nil {
		in, out := &in.Children, &out.Children
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.OutboundNodes != nil {
		in, out := &in.OutboundNodes, &out.OutboundNodes
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeStatus.
func (in *NodeStatus) DeepCopy() *NodeStatus {
	if in == nil {
		return nil
	}
	out := new(NodeStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NoneStrategy) DeepCopyInto(out *NoneStrategy) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NoneStrategy.
func (in *NoneStrategy) DeepCopy() *NoneStrategy {
	if in == nil {
		return nil
	}
	out := new(NoneStrategy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Outputs) DeepCopyInto(out *Outputs) {
	*out = *in
	if in.Parameters != nil {
		in, out := &in.Parameters, &out.Parameters
		*out = make([]Parameter, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Artifacts != nil {
		in, out := &in.Artifacts, &out.Artifacts
		*out = make([]Artifact, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Result != nil {
		in, out := &in.Result, &out.Result
		*out = new(string)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Outputs.
func (in *Outputs) DeepCopy() *Outputs {
	if in == nil {
		return nil
	}
	out := new(Outputs)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Parameter) DeepCopyInto(out *Parameter) {
	*out = *in
	if in.Default != nil {
		in, out := &in.Default, &out.Default
		*out = new(string)
		**out = **in
	}
	if in.Value != nil {
		in, out := &in.Value, &out.Value
		*out = new(string)
		**out = **in
	}
	if in.ValueFrom != nil {
		in, out := &in.ValueFrom, &out.ValueFrom
		*out = new(ValueFrom)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Parameter.
func (in *Parameter) DeepCopy() *Parameter {
	if in == nil {
		return nil
	}
	out := new(Parameter)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PodGC) DeepCopyInto(out *PodGC) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PodGC.
func (in *PodGC) DeepCopy() *PodGC {
	if in == nil {
		return nil
	}
	out := new(PodGC)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RawArtifact) DeepCopyInto(out *RawArtifact) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RawArtifact.
func (in *RawArtifact) DeepCopy() *RawArtifact {
	if in == nil {
		return nil
	}
	out := new(RawArtifact)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceTemplate) DeepCopyInto(out *ResourceTemplate) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceTemplate.
func (in *ResourceTemplate) DeepCopy() *ResourceTemplate {
	if in == nil {
		return nil
	}
	out := new(ResourceTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RetryStrategy) DeepCopyInto(out *RetryStrategy) {
	*out = *in
	if in.Limit != nil {
		in, out := &in.Limit, &out.Limit
		*out = new(int32)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RetryStrategy.
func (in *RetryStrategy) DeepCopy() *RetryStrategy {
	if in == nil {
		return nil
	}
	out := new(RetryStrategy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *S3Artifact) DeepCopyInto(out *S3Artifact) {
	*out = *in
	in.S3Bucket.DeepCopyInto(&out.S3Bucket)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new S3Artifact.
func (in *S3Artifact) DeepCopy() *S3Artifact {
	if in == nil {
		return nil
	}
	out := new(S3Artifact)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *S3Bucket) DeepCopyInto(out *S3Bucket) {
	*out = *in
	if in.Insecure != nil {
		in, out := &in.Insecure, &out.Insecure
		*out = new(bool)
		**out = **in
	}
	in.AccessKeySecret.DeepCopyInto(&out.AccessKeySecret)
	in.SecretKeySecret.DeepCopyInto(&out.SecretKeySecret)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new S3Bucket.
func (in *S3Bucket) DeepCopy() *S3Bucket {
	if in == nil {
		return nil
	}
	out := new(S3Bucket)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ScriptTemplate) DeepCopyInto(out *ScriptTemplate) {
	*out = *in
	in.Container.DeepCopyInto(&out.Container)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ScriptTemplate.
func (in *ScriptTemplate) DeepCopy() *ScriptTemplate {
	if in == nil {
		return nil
	}
	out := new(ScriptTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Sequence) DeepCopyInto(out *Sequence) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Sequence.
func (in *Sequence) DeepCopy() *Sequence {
	if in == nil {
		return nil
	}
	out := new(Sequence)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SuspendTemplate) DeepCopyInto(out *SuspendTemplate) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SuspendTemplate.
func (in *SuspendTemplate) DeepCopy() *SuspendTemplate {
	if in == nil {
		return nil
	}
	out := new(SuspendTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TarStrategy) DeepCopyInto(out *TarStrategy) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TarStrategy.
func (in *TarStrategy) DeepCopy() *TarStrategy {
	if in == nil {
		return nil
	}
	out := new(TarStrategy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Template) DeepCopyInto(out *Template) {
	*out = *in
	in.Arguments.DeepCopyInto(&out.Arguments)
	if in.TemplateRef != nil {
		in, out := &in.TemplateRef, &out.TemplateRef
		*out = new(TemplateRef)
		**out = **in
	}
	in.Inputs.DeepCopyInto(&out.Inputs)
	in.Outputs.DeepCopyInto(&out.Outputs)
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Affinity != nil {
		in, out := &in.Affinity, &out.Affinity
		*out = new(v1.Affinity)
		(*in).DeepCopyInto(*out)
	}
	in.Metadata.DeepCopyInto(&out.Metadata)
	if in.Daemon != nil {
		in, out := &in.Daemon, &out.Daemon
		*out = new(bool)
		**out = **in
	}
	if in.Steps != nil {
		in, out := &in.Steps, &out.Steps
		*out = make([][]WorkflowStep, len(*in))
		for i := range *in {
			if (*in)[i] != nil {
				in, out := &(*in)[i], &(*out)[i]
				*out = make([]WorkflowStep, len(*in))
				for i := range *in {
					(*in)[i].DeepCopyInto(&(*out)[i])
				}
			}
		}
	}
	if in.Container != nil {
		in, out := &in.Container, &out.Container
		*out = new(v1.Container)
		(*in).DeepCopyInto(*out)
	}
	if in.Script != nil {
		in, out := &in.Script, &out.Script
		*out = new(ScriptTemplate)
		(*in).DeepCopyInto(*out)
	}
	if in.Resource != nil {
		in, out := &in.Resource, &out.Resource
		*out = new(ResourceTemplate)
		**out = **in
	}
	if in.DAG != nil {
		in, out := &in.DAG, &out.DAG
		*out = new(DAGTemplate)
		(*in).DeepCopyInto(*out)
	}
	if in.Suspend != nil {
		in, out := &in.Suspend, &out.Suspend
		*out = new(SuspendTemplate)
		**out = **in
	}
	if in.Volumes != nil {
		in, out := &in.Volumes, &out.Volumes
		*out = make([]v1.Volume, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.InitContainers != nil {
		in, out := &in.InitContainers, &out.InitContainers
		*out = make([]UserContainer, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Sidecars != nil {
		in, out := &in.Sidecars, &out.Sidecars
		*out = make([]UserContainer, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.ArchiveLocation != nil {
		in, out := &in.ArchiveLocation, &out.ArchiveLocation
		*out = new(ArtifactLocation)
		(*in).DeepCopyInto(*out)
	}
	if in.ActiveDeadlineSeconds != nil {
		in, out := &in.ActiveDeadlineSeconds, &out.ActiveDeadlineSeconds
		*out = new(int64)
		**out = **in
	}
	if in.RetryStrategy != nil {
		in, out := &in.RetryStrategy, &out.RetryStrategy
		*out = new(RetryStrategy)
		(*in).DeepCopyInto(*out)
	}
	if in.Parallelism != nil {
		in, out := &in.Parallelism, &out.Parallelism
		*out = new(int64)
		**out = **in
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Priority != nil {
		in, out := &in.Priority, &out.Priority
		*out = new(int32)
		**out = **in
	}
	if in.HostAliases != nil {
		in, out := &in.HostAliases, &out.HostAliases
		*out = make([]v1.HostAlias, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.SecurityContext != nil {
		in, out := &in.SecurityContext, &out.SecurityContext
		*out = new(v1.PodSecurityContext)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Template.
func (in *Template) DeepCopy() *Template {
	if in == nil {
		return nil
	}
	out := new(Template)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *TemplateRef) DeepCopyInto(out *TemplateRef) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TemplateRef.
func (in *TemplateRef) DeepCopy() *TemplateRef {
	if in == nil {
		return nil
	}
	out := new(TemplateRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *UserContainer) DeepCopyInto(out *UserContainer) {
	*out = *in
	in.Container.DeepCopyInto(&out.Container)
	if in.MirrorVolumeMounts != nil {
		in, out := &in.MirrorVolumeMounts, &out.MirrorVolumeMounts
		*out = new(bool)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UserContainer.
func (in *UserContainer) DeepCopy() *UserContainer {
	if in == nil {
		return nil
	}
	out := new(UserContainer)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ValueFrom) DeepCopyInto(out *ValueFrom) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ValueFrom.
func (in *ValueFrom) DeepCopy() *ValueFrom {
	if in == nil {
		return nil
	}
	out := new(ValueFrom)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Workflow) DeepCopyInto(out *Workflow) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Workflow.
func (in *Workflow) DeepCopy() *Workflow {
	if in == nil {
		return nil
	}
	out := new(Workflow)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Workflow) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkflowList) DeepCopyInto(out *WorkflowList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Workflow, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkflowList.
func (in *WorkflowList) DeepCopy() *WorkflowList {
	if in == nil {
		return nil
	}
	out := new(WorkflowList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *WorkflowList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkflowSpec) DeepCopyInto(out *WorkflowSpec) {
	*out = *in
	if in.Templates != nil {
		in, out := &in.Templates, &out.Templates
		*out = make([]Template, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.Arguments.DeepCopyInto(&out.Arguments)
	if in.Volumes != nil {
		in, out := &in.Volumes, &out.Volumes
		*out = make([]v1.Volume, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.VolumeClaimTemplates != nil {
		in, out := &in.VolumeClaimTemplates, &out.VolumeClaimTemplates
		*out = make([]v1.PersistentVolumeClaim, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Parallelism != nil {
		in, out := &in.Parallelism, &out.Parallelism
		*out = new(int64)
		**out = **in
	}
	if in.ArtifactRepositoryRef != nil {
		in, out := &in.ArtifactRepositoryRef, &out.ArtifactRepositoryRef
		*out = new(ArtifactRepositoryRef)
		**out = **in
	}
	if in.Suspend != nil {
		in, out := &in.Suspend, &out.Suspend
		*out = new(bool)
		**out = **in
	}
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Affinity != nil {
		in, out := &in.Affinity, &out.Affinity
		*out = new(v1.Affinity)
		(*in).DeepCopyInto(*out)
	}
	if in.Tolerations != nil {
		in, out := &in.Tolerations, &out.Tolerations
		*out = make([]v1.Toleration, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.ImagePullSecrets != nil {
		in, out := &in.ImagePullSecrets, &out.ImagePullSecrets
		*out = make([]v1.LocalObjectReference, len(*in))
		copy(*out, *in)
	}
	if in.HostNetwork != nil {
		in, out := &in.HostNetwork, &out.HostNetwork
		*out = new(bool)
		**out = **in
	}
	if in.DNSPolicy != nil {
		in, out := &in.DNSPolicy, &out.DNSPolicy
		*out = new(v1.DNSPolicy)
		**out = **in
	}
	if in.DNSConfig != nil {
		in, out := &in.DNSConfig, &out.DNSConfig
		*out = new(v1.PodDNSConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.TTLSecondsAfterFinished != nil {
		in, out := &in.TTLSecondsAfterFinished, &out.TTLSecondsAfterFinished
		*out = new(int32)
		**out = **in
	}
	if in.ActiveDeadlineSeconds != nil {
		in, out := &in.ActiveDeadlineSeconds, &out.ActiveDeadlineSeconds
		*out = new(int64)
		**out = **in
	}
	if in.Priority != nil {
		in, out := &in.Priority, &out.Priority
		*out = new(int32)
		**out = **in
	}
	if in.PodGC != nil {
		in, out := &in.PodGC, &out.PodGC
		*out = new(PodGC)
		**out = **in
	}
	if in.PodPriority != nil {
		in, out := &in.PodPriority, &out.PodPriority
		*out = new(int32)
		**out = **in
	}
	if in.HostAliases != nil {
		in, out := &in.HostAliases, &out.HostAliases
		*out = make([]v1.HostAlias, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.SecurityContext != nil {
		in, out := &in.SecurityContext, &out.SecurityContext
		*out = new(v1.PodSecurityContext)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkflowSpec.
func (in *WorkflowSpec) DeepCopy() *WorkflowSpec {
	if in == nil {
		return nil
	}
	out := new(WorkflowSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkflowStatus) DeepCopyInto(out *WorkflowStatus) {
	*out = *in
	in.StartedAt.DeepCopyInto(&out.StartedAt)
	in.FinishedAt.DeepCopyInto(&out.FinishedAt)
	if in.Nodes != nil {
		in, out := &in.Nodes, &out.Nodes
		*out = make(map[string]NodeStatus, len(*in))
		for key, val := range *in {
			(*out)[key] = *val.DeepCopy()
		}
	}
	if in.PersistentVolumeClaims != nil {
		in, out := &in.PersistentVolumeClaims, &out.PersistentVolumeClaims
		*out = make([]v1.Volume, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Outputs != nil {
		in, out := &in.Outputs, &out.Outputs
		*out = new(Outputs)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkflowStatus.
func (in *WorkflowStatus) DeepCopy() *WorkflowStatus {
	if in == nil {
		return nil
	}
	out := new(WorkflowStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkflowStep) DeepCopyInto(out *WorkflowStep) {
	*out = *in
	in.Arguments.DeepCopyInto(&out.Arguments)
	if in.TemplateRef != nil {
		in, out := &in.TemplateRef, &out.TemplateRef
		*out = new(TemplateRef)
		**out = **in
	}
	if in.WithItems != nil {
		in, out := &in.WithItems, &out.WithItems
		*out = make([]Item, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.WithSequence != nil {
		in, out := &in.WithSequence, &out.WithSequence
		*out = new(Sequence)
		**out = **in
	}
	if in.ContinueOn != nil {
		in, out := &in.ContinueOn, &out.ContinueOn
		*out = new(ContinueOn)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkflowStep.
func (in *WorkflowStep) DeepCopy() *WorkflowStep {
	if in == nil {
		return nil
	}
	out := new(WorkflowStep)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkflowTemplate) DeepCopyInto(out *WorkflowTemplate) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkflowTemplate.
func (in *WorkflowTemplate) DeepCopy() *WorkflowTemplate {
	if in == nil {
		return nil
	}
	out := new(WorkflowTemplate)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *WorkflowTemplate) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkflowTemplateList) DeepCopyInto(out *WorkflowTemplateList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]WorkflowTemplate, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkflowTemplateList.
func (in *WorkflowTemplateList) DeepCopy() *WorkflowTemplateList {
	if in == nil {
		return nil
	}
	out := new(WorkflowTemplateList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *WorkflowTemplateList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WorkflowTemplateSpec) DeepCopyInto(out *WorkflowTemplateSpec) {
	*out = *in
	if in.Templates != nil {
		in, out := &in.Templates, &out.Templates
		*out = make([]Template, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.Arguments.DeepCopyInto(&out.Arguments)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WorkflowTemplateSpec.
func (in *WorkflowTemplateSpec) DeepCopy() *WorkflowTemplateSpec {
	if in == nil {
		return nil
	}
	out := new(WorkflowTemplateSpec)
	in.DeepCopyInto(out)
	return out
}
