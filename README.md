# Note: kind instead of k3d

This is because we need dpkg installed for the blob-csi-driver, and `k3d` is too minimal for that.

# Three S3Proxy instances

```
s3proxy
|- standard
|- premium
`- fdi-unclassified

s3proxy-ro
|- standard-ro
|- premium-ro
`- fdi-unclassified-ro

s3proxy-prob
|- prob
`- fdi-prob
```

# Architecture

The architecture has a nice split:

- Create or Find Storage Containers, Create PVs (readwrite or readonly), and create PVCs in the user namespace

- Automatically discover PVCs and bind them to Notebooks, Workflows, or S3Proxy pods. (Goofys Webhook)

**Refreshes/Updates**

- Users will need to restart their notebooks to trigger refresh of mounts (in the situation where an FDI bucket gets added, for example).

- Recommend that S3Proxy comes with a CronJob that rolls out a restart on an interval. (Similar to the AAD Pod refresher.)

**TODO: What happens if the storage account credentials get rotated?**

## BlobCSI controller

The BlobCSI Profile Controller loops through profiles and AAW Storage accounts:

- Creates a container (bucket) in each storage account if it doesn't exist.
- Creates a PersistentVolume binding to it using the Azure Blob CSI Driver. The driver authenticates using a secret within the `azure-blob-csi-system`
- Creates a PersistentVolumeClaim binding to the PersistentVolume in the profile's namespace.[^A Gatekeeper policy is in place to ensure that no other PVCs are allowed to bind to this PV]

### FDI Submodule

Use OPA sdk to check if a user has access to a given bucket, and determine what permissions they have. (We can only implement RW/RO)

## Gatekeeper Policies

The PersistentVolumes are created with a `profile` label, matching the users profile. A Gatekeeper policy ensure that `pvc.metadata.namespace == pv.metadata.labels.profile`. So that users cannot bind other users volumes.

Also, the classification of the PV and PVCs must match. 

**TODO: Check if the PV/PVC classification logic is already covered.**

## Goofys Injector

The Goofys injector is repurposed:

- Instead of using a fixed list of mounts, it uses the `blob.aaw.statcan.gc.ca/automount` label to select volumes to mount.
- It further differentiates between `unclassified` and `protected-b` PVCs, only mounting to the correct pods.
