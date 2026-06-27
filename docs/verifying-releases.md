# Verifying release binaries

Every `spotctl` release artifact (each archive and the `checksums.txt`) is signed with
[cosign](https://docs.sigstore.dev/cosign/overview/) using the project key. The public key lives at
[`cosign.pub`](../cosign.pub) in the repository root.

For a downloaded artifact `spotctl_<version>_<os>_<arch>.tar.gz` and its `.sig` from the release page:

```sh
cosign verify-blob \
  --key cosign.pub \
  --signature spotctl_<version>_<os>_<arch>.tar.gz.sig \
  spotctl_<version>_<os>_<arch>.tar.gz
```

A successful check prints `Verified OK`.

You can also verify the checksum file itself, then check the binary's SHA256 against it:

```sh
cosign verify-blob --key cosign.pub --signature checksums.txt.sig checksums.txt
sha256sum --check --ignore-missing checksums.txt
```
