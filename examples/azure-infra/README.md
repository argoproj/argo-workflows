## Azure Infra Example

Use Azure CLI to:
- Create VM
- Perform Installs via Ansible
- Other tests / builds, if applicable
- Delete VM

Requires:
- Azure subscription and resource group
- Azure Service Principal (SP) details, including client secret
- Pre-configured vNet and subnet

Usage sequence: 
- Generate K8s secret via kubectl command in `create-az-sp-secret.cmd.example`. Edit SP details, and desired VM admin password. 
- Apply all WorkflowTemplates. Edit Azure subscription, resource group, vNet, subnet. 
- Run `azure-vm-lifecycle` to execute end-to-end process. 
