---
id: gcp_kms
title: GCPKMS
---

# **Google Cloud Platform configuration**

Create a new project or use an existing project and the service accounts used with Signatory should have the following permissions. It may be achieved by using custom roles (see [https://console.cloud.google.com/iam-admin/roles](https://console.cloud.google.com/iam-admin/roles)) \
Project name is required in the signatory config.

## **Basic permissions**

* `cloudkms.cryptoKeyVersions.get`
* `cloudkms.cryptoKeyVersions.list`
* `cloudkms.cryptoKeyVersions.viewPublicKey`
* `cloudkms.cryptoKeys.get`
* `cloudkms.cryptoKeys.list`

## **Sign**

* `cloudkms.cryptoKeyVersions.useToSign`

## **Import**

* `cloudkms.cryptoKeyVersions.create`
* `cloudkms.cryptoKeys.create`
* `cloudkms.importJobs.create`
* `cloudkms.importJobs.get`
* `cloudkms.importJobs.list`
* `cloudkms.importJobs.useToImport`

## **Configuration parameters**

Below are the configuration fields which are required for Signatory.

|||||
|--- |--- |--- |--- |
|Name|Type|Required|Description|
|application_credentials|string|OPTIONAL|Path to the GCP application token JSON file (overrides GOOGLE_APPLICATION_CREDENTIALS environment variable)|
|application_credentials_data|string|OPTIONAL|GCP application token JSON data (overrides application_credentials)|
|project|string|✅|Project name|
|location|string|✅|Location|
|key_ring|string|✅|Key ring name|

## **Key Management**

Under `key management` create a new `key-ring` with any location and create a key with `purpose` as `Asymmetric-sign` and `protection level` as `HSM`.

The key-ring name and location are required in the signatory configuration.

## **Application Access:**

The below steps are for providing signatory with the permissions to access the google cloud account Key Management.

* Select `IAM & ADMIN` from the menu and select `Service accounts`. Create a new service account or use an existing one with all the above permissions (Get, Sign & Import) granted.
* Select the created/existing service account and within that create a new key and a prompt to download the application credentials will appear, select the JSON format.
* The downloaded JSON file is needed in signatory config or can be assigned to the below environment variable.

## **Environment variables**

`cloudkms` backend accepts GCP's standard `GOOGLE_APPLICATION_CREDENTIALS` environment variable

```sh
export GOOGLE_APPLICATION_CREDENTIALS="signatory-testing-a7sdfew625aecb.json"
```

## **Getting PKH**

```sh
signatory % ./signatory-cli list -c /etc/s.yaml
Public Key Hash:    tz3fK7rVYSg2HTEAmUYdfjJWSDGfsKrxH3xQ
Vault:              CloudKMS
ID:                 projects/signatory-testing/locations/europe-north1/keyRings/sigy-key/cryptoKeys/sigyhsm/cryptoKeyVersions/4
Status:             FOUND_NOT_CONFIGURED
*DISABLED*
```

**Update signatory.yaml config with PKH:**

```sh
signatory % cat /etc/s.yaml 
server:
  # Address/Port that Signatory listens on
  address: :6732
  # Address/Port that Signatory serves prometheus metrics on
  utility_address: :9583

vaults:
  kms:
    driver: cloudkms
    config:
      project: signatory-testing
      location: europe-south1
      key_ring: sigy-key
tezos:
  # Default policy allows "block" and "endorsement" operations
  tz3fK7rVYSg2HTEAmUYdfjJWSDGfsKrxH3xQ:
    # Setting `log_payloads` to `true` will cause Signatory to log operation
    # payloads to `stdout`. This may be desirable for audit and investigative
    # purposes.
    log_payloads: true
    allowed_operations:
      # List of [generic, block, endorsement]
      - generic
      - block
      - endorsement
    allowed_kinds:
      # List of [endorsement, ballot, reveal, transaction, origination, delegation, seed_nonce_revelation, activate_account]
      - transaction
      - endorsement
```

## **Key Import:**

Users can generate a private key in an air gap environment and then import it into GCP Key Management using `signatory-cli` binary. Below are the steps to do that.

1. Build `signatory-cli` binary using `make signatory-cli`. You need `Golang version 1.15` or later.

2. Use the below command to import the generated private into GCP Key Management. Only `Elliptic Curve P-256 - SHA256` `Digest` is supported now. Below sample key is taken from `signatory/docs/yubihsm.md`

```sh
% ./signatory-cli import -c signatory.yaml --vault kms p2esk28hoUE2J88QNFj2aDX2pjzL7wcVh2g8tkEwtWWguby9M3FHUgSbzvF2Sd7wQ4Kd8crFwvto6gF3otcBuo4T

INFO[0000] Initializing vault                            vault=cloudkms vault_name=kms
Enter Password: INFO[0002] Requesting import operation                   pkh=tz3be5v4ZWL3zQYUZoLWJQy8P3H6RJryVVXn vault=CloudKMS vault_name=projects/signatory-testing/locations/europe-north1/keyRings/sign-ring
INFO[0008] Successfully imported                         key_id=projects/signatory-testing/locations/europe-north1/keyRings/sign-ring/cryptoKeys/signatory-imported-215FwcXxhLdlr9IYwzA31vwANmy/cryptoKeyVersions/1 pkh=tz3be5v4ZWL3zQYUZoLWJQy8P3H6RJryVVXn vault=CloudKMS vault_name=projects/signatory-testing/locations/europe-north1/keyRings/sign-ring
