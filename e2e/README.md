# End to End Tests

This package contains integration tests. Unlike tests alongside Nomad code,
these tests expect there to already be a functional Nomad cluster accessible
(either on localhost or via the `NOMAD_ADDR` env var).

The `NOMAD_E2E=1` environment variable must be set for these tests to run.

## Local Nomad Development

When developing tests locally, provisioning is not required when only the tests
change. See [`framework/doc.go`](framework/doc.go) for how to write tests.

When making changes to the Nomad agent itself, use `./bin/update $(which nomad)
/usr/local/bin/nomad` and `./bin/run sudo systemctl restart nomad` to
destructively modify the provisioned cluster.

## Provisioning Test Infrastructure on AWS

The `terraform/` folder has provisioning code to spin up a Nomad cluster on
AWS. You'll need both Terraform and AWS credentials to setup AWS instances on
which e2e tests will run. See the
[README](https://github.com/hashicorp/nomad/blob/master/e2e/terraform/README.md)
for details. The number of servers and clients is configurable, as is the
specific build of Nomad to deploy and the configuration file for each client
and server.

## Running

After completing the provisioning step above, you can set the client
environment for `NOMAD_ADDR` and run the tests as shown below:

```sh
# from the ./e2e/terraform directory, set your client environment
# if you haven't already
$(terraform output environment)

cd ..
go test -v .
```

If you want to run a specific suite, you can specify the `-suite` flag as
shown below. Only the suite with a matching `Framework.TestSuite.Component`
will be run, and all others will be skipped.

```sh
go test -v -suite=Consul .
```

If you want to run a specific test, you'll need to regex-escape some of the
test's name so that the test runner doesn't skip over framework struct method
names in the full name of the tests:

```sh
go test -v . -run 'TestE2E/Consul/\*consul\.ScriptChecksE2ETest/TestGroup'
                              ^       ^             ^               ^
                              |       |             |               |
                          Component   |             |           Test func
                                      |             |
                                  Go Package      Struct
```

## I Want To...

### ...SSH Into One Of The Test Machines

You can use the Terraform output to find the IP address. The keys will
in the `./terraform/keys/` directory.

```sh
ssh -i keys/nomad-e2e-*.pem ubuntu@${EC2_IP_ADDR}
```

Run `terraform output` for IP addresses and details.

### ...Deploy a Cluster of Mixed Nomad Versions

The `variables.tf` file describes the `nomad_sha`, `nomad_version`, and
`nomad_local_binary` variable that can be used for most circumstances. But if
you want to deploy mixed Nomad versions, you can provide a list of versions in
your `terraform.tfvars` file.

For example, if you want to provision 3 servers all using Nomad 0.12.1, and 2
Linux clients using 0.12.1 and 0.12.2, you can use the following variables:

```hcl
# will be used for servers
nomad_version = "0.12.1"

# will override the nomad_version for Linux clients
nomad_version_client_linux = [
    "0.12.1",
    "0.12.2"
]
```

### ...Deploy Custom Configuration Files

Set the `profile` field to `"custom"` and put the configuration files in
`./terraform/config/custom/` as described in the
[README](https://github.com/hashicorp/nomad/blob/master/e2e/terraform/README.md#Profiles).

### ...Deploy More Than 4 Linux Clients

Use the `"custom"` profile as described above.

### ...Change the Nomad Version After Provisioning

You can update the `nomad_sha` or `nomad_version` variables, or simply rebuild
the binary you have at the `nomad_local_binary` path so that Terraform picks
up the changes. Then run `terraform plan`/`terraform apply` again. This will
update Nomad in place, making the minimum amount of changes neccessary.
