from kapitan.inputs import kadet
import yaml

inv = kadet.inventory()

class User(kadet.BaseObj):
    def new(self):
        self.need("username")
        self.need("personas")

    def body(self):
        self.root.metadata.labels = {"neo4j.crossplane.io": self.username}

obj = User(
    username=inv.parameters.user.username,
    personas=inv.parameters.user.personas
)

def main():
    return yaml.dump(obj.dump())