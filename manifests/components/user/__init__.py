from kapitan.inputs.kadet import BaseObj
from kapitan.inputs import kadet

inv = kadet.inventory()

class User(BaseObj):
    def new(self):
        self.need("api_version")
        self.need("username")
        self.need("personas")

    def body(self):
        self.root.apiVersion = "powerbroker.neo4j.crossplane.io/" + self.kwargs.api_version
        self.root.kind = "User"
        self.set_spec(self.kwargs.username, self.kwargs.personas)
        self.set_metadata()
    
    def set_metadata(self):
        self.root.metadata.labels = inv.parameters.metadata.labels

    def set_spec(self, username, personas):
        self.root.spec.forProvider.username = username
        self.root.spec.forProvider.personas = personas

def main():
    obj = BaseObj()
    obj.root.user = User(
        api_version=inv.parameters.api_version,
        username=inv.parameters.username,
        personas=inv.parameters.personas
    )

    return obj