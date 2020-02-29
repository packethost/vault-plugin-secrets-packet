# Vault Secrets Engine plugin for Packet

This is a [custom Vault secrets engine plugin](https://www.vaultproject.io/docs/plugin/). 
When installed in Vault, it allows to create temporary API keys in the Packet API. 
Vault then manages creation and removal of requested credentials.

[Vault](https://www.vaultproject.io/) from [HashiCorp](https://www.hashicorp.com/)
allows you to secure, store and tightly control access to tokens, passwords, 
certificates, encryption keys for protecting secrets and other sensitive data using a UI, CLI, or HTTP API.

[Packet](https://www.packet.com/) is a bare metal cloud provider
with an [API](https://www.packet.com/developers/api/) for provisioning
and managing resources. You will need a Packet account to use this plugin.

## Installation

### Download
Download latest release for your arch from the [release page](https://github.com/packethost/vault-plugin-secrets-packet/releases).

### Build yourself

- Clone this repo
- Run `make bootstrap` and `make dev`
- Locate the directory where the binary `vault-plugin-secrets-packet` lives. It should be `./bin/` under this repo, e.g. `/home/tomk/vault-plugin-secrets-packet/bin`
- [Download and install Vault](https://www.vaultproject.io/downloads/)
- In your [vault config](https://www.vaultproject.io/docs/configuration/), specify `plugin_directory = "/home/tomk/vault-plugin-secrets-packet/bin"` in the top scope
- Run vault and pass the path to your config file, e.g. `vault server -config=/home/tomk/vdir/config.hcl`
- Enable the plugin (substitute paths to your own):

```
$ vault write sys/plugins/catalog/secret/packet \
        sha_256="$(shasum -a 256 /home/.../bin/vault-plugin-secrets-packet | cut -d " " -f1)" \
        command="vault-plugin-secrets-packet"

$ vault secrets enable --plugin-name='packet' --path="packet" plugin    
```

### Dev setup

Vault needs a storage back end and maybe it's too much work for you 
to install a Consul cluster for testing. Fortunately, Vault server 
supports "Development mode". You can get by with `config.hcl` as just:

```
plugin_directory = "/home/tomk/vault-plugin-secrets-packet/bin"
```

.. if you run vault server as 

```
vault server -dev -config=./config.hcl
```

Vault API in the dev mode listens on `127.0.0.1:8200` by default, you should do 
```
export VAULT_ADDR='http://127.0.0.1:8200'
```
.. before any of the other vault commands.


## Usage

In order to use the Packet secrets engine, you need to configure it with a user read-write API key:

```
$ vault kv put packet/config api_token=$PACKET_AUTH_TOKEN
```

That API key will be used to create and destroy the Vault-managed API keys.


### Create a role for getting user read-only API tokens with 30s TTL

To create a user role with given parameters, do

```
$ vault kv put packet/role/userrole type=user read_only=true ttl=30 max_ttl=30
```

Then you can get temporary credentials for this role (with this parameters):

```
$ vault kv get packet/creds/userrole
```

### Create a role for gettting project read-only API tokens with 30s TTL

To create a role for given project, do:

```
vault kv put packet/role/projectrole \
      type=project \
      project_id=52634fb2-ee46-4673-242a-de2c2bdba33b \
      read_only=true \
      ttl=30 \
      max_ttl=30
```

.. and check the parameters of the role

```
$ vault kv get packet/role/projectrole

======= Data =======
Key           Value
---           -----
max_ttl       30s
project_id    52634fb2-ee46-4673-242a-de2c2bdba33b
read_only     true
ttl           30s
type          project

```

Then, you can get temporary API token for the project as:

```
$ vault kv get packet/creds/projectrole

======== Data ========
Key              Value
---              -----
api_key_token    gsdrDRGrEGRSGDRGdgrgdrgdrgDrgdg3

```

## Examples

[Mayday](https://github.com/grahamc/mayday) is using Vault-generated ephemeral secrets to pipe output of SOS consoles from device in Packet to a local file.

The plugin can also be simply used with curl: 

```
export VAULT_ADDR='http://127.0.0.1:8200'

# first create a role
vault kv put packet/role/short-read-only-user \
    type=user read_only=true ttl=30 max_ttl=3600

# query projects using Vault-generated credentials
curl \
     --header 'Accept: application/json' \
     --header 'Content-Type: application/json' \
     --header "X-Auth-Token: $(vault kv get -field=api_key_token packet/creds/short-read-only-user)" \
     https://api.packet.net/projects
``` 
