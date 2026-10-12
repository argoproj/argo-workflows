"""Real server save/history/model update/semantic Diff; no execution side effects."""
import json,uuid,copy
from e2e import api,PREFIX,NS
from template_refs_e2e import apply,resource,native,kubectl
RUN='diff-'+uuid.uuid4().hex[:8]
def main():
 model={'dialect':'http://json-schema.org/draft-07/schema#','schema':{'type':'object','properties':{'enabled':{'type':'boolean','default':True},'replicas':{'type':'integer','default':2},'items':{'type':'array','items':{'type':'object','properties':{'id':{'type':'string'}},'required':['id']}}},'additionalProperties':False},'identityRules':[{'path':'/items','keyPath':'/id'}]}
 apply(resource('ParameterTemplate',RUN,model))
 apply(native('WorkflowTemplate',RUN,{'entrypoint':'show','serviceAccountName':'workflow-executor','arguments':{'parameters':[{'name':'payload'}]},'templates':[{'name':'show','container':{'image':'alpine:3.22','command':['echo','{{workflow.parameters.payload}}']}}]}))
 apply(resource('WorkflowParameterBinding',RUN,{'workflowTemplate':RUN,'parameterTemplate':RUN,'mappings':[{'path':'','parameter':'payload','encoding':'json'}]}))
 before={o['metadata']['uid'] for o in json.loads(kubectl('-n',NS,'get','workflows','-o','json'))['items']}
 one=api('PUT',PREFIX+'/sets/'+RUN,{'template':RUN,'values':{'items':[{'id':'east'},{'id':'west'}]},'resourceVersion':''});r1=one['spec']['revision']
 two=api('PUT',PREFIX+'/sets/'+RUN,{'template':RUN,'values':{'items':[{'id':'west'},{'id':'east'}],'enabled':False,'replicas':0},'resourceVersion':one['metadata']['resourceVersion']});r2=two['spec']['revision']
 comparison=api('POST',PREFIX+'/diff',{'left':r1,'right':r2});assert any(c['op']=='move' and c['itemIdentity']=='west' for c in comparison['changes']);assert any(c['path']=='/enabled' and c['after'] is False for c in comparison['changes']);assert any(c['path']=='/replicas' and c['after']==0 for c in comparison['changes']);assert comparison['sourceChanges'];assert not comparison['modelChanged']
 model['identityRules']=[];model['schema']['properties']['replicas']['default']=3;apply(resource('ParameterTemplate',RUN,model))
 three=api('PUT',PREFIX+'/sets/'+RUN,{'template':RUN,'values':{'items':[{'id':'east'},{'id':'west'}],'enabled':False,'replicas':0},'resourceVersion':two['metadata']['resourceVersion']});r3=three['spec']['revision']
 changed=api('POST',PREFIX+'/diff',{'left':r2,'right':r3});assert changed['identityRulesChanged'] and changed['modelChanged'] and changed['modelChanges'];assert any(c['path']=='/items' and c['op']=='replace' for c in changed['changes'])
 # Directly imported revisions must not leak plaintext if either model declares it sensitive.
 imported=copy.deepcopy(api('GET',PREFIX+'/catalog/parameterrevisions?name='+r1)['spec']);imported['values']={'credential':'synthetic-sensitive-marker'};imported['errors']=[]
 apply(resource('ParameterRevision',RUN+'-unsafe',imported))
 declared=copy.deepcopy(imported);declared['model']['sensitivePaths']=['/credential'];declared['values']={'credential':{'secretRef':{'name':'synthetic','key':'value'}}}
 apply(resource('ParameterRevision',RUN+'-sensitive',declared))
 for left,right in [('unsafe','sensitive'),('sensitive','unsafe')]:
  rejected=api('POST',PREFIX+'/diff',{'left':RUN+'-'+left,'right':RUN+'-'+right},400)
  assert 'synthetic-sensitive-marker' not in json.dumps(rejected)
 history=api('GET',PREFIX+'/history/'+RUN);assert len(history['items'])==3
 assert before=={o['metadata']['uid'] for o in json.loads(kubectl('-n',NS,'get','workflows','-o','json'))['items']}
 print(json.dumps({'run':RUN,'revisions':[r1,r2,r3],'firstComparison':comparison,'modelComparison':changed,'workflowWrites':0}),flush=True)
if __name__=='__main__':main()
