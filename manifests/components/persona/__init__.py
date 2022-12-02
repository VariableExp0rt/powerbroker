from kapitan.inputs.kadet import BaseObj
from kapitan.inputs import kadet

inv = kadet.inventory()

class Persona(BaseObj):
    def new(self):
        self.need("api_version")
        self.need("name")
        self.need("permission_set_refs")

    def body(self):
        self.root.apiVersion = "powerbroker.neo4j.crossplane.io/" + self.kwargs.api_version
        self.root.kind = "Persona"
        self.set_spec(self.kwargs.name, self.kwargs.permissionSet_refs)
        self.set_metadata()
    
    def set_metadata(self):
        self.root.metadata.name = inv.parameters.name
        self.root.metadata.labels = inv.parameters.metadata.labels

    def set_spec(self, name, permission_set_refs):
        self.root.spec.forProvider.name = name
        self.root.spec.forProvider.permissionSetRefs = permission_set_refs
        self.root.spec.providerConfigRef.name = inv.parameters.provider_config_ref

def main():
    obj = BaseObj()
    obj.root.user = Persona(
        api_version=inv.parameters.api_version,
        name=inv.parameters.name,
        permissionSet_refs=inv.parameters.permissionSetRefs
    )

    return obj