# Bounded offline regression selection
Unset CLOUDSIGMA_USERNAME, CLOUDSIGMA_PASSWORD, CLOUDSIGMA_TOKEN and CLOUDSIGMA_LOCATION.
With an already populated checksum-verified Go module cache:

    GOPROXY=off GOSUMDB=off TF_ACC= go test ./cloudsigma ./internal/provider ./readonly -run '^(TestOffline|TestFence|TestStructure|TestResourceCloudSigmaServer_findIPv4Address|TestFilter)' -count=1 -v

Fixtures use synthetic inputs and in-memory HTTP transports. No Terraform CLI or cloud resource is required. Do not substitute unrestricted `go test ./...`: upstream framework unit tests can bootstrap Terraform CLI even without TF_ACC. This transport boundary is not OS-level network/process isolation.

Snapshot create/read/update/delete logging is exercised against fake HTTP, including four expected calls for create. Credential, profile and VNC coverage is bounded; Sensitive does not encrypt persisted state, and SDK error bodies can remain in diagnostics. The readonly test support is not unrestricted-engine deployment approval. Source changes remove provider-owned object log fields while retaining phase messages, upgrade the SDK empty-clone fix and mark legacy VNC password sensitive. No live lifecycle or release claim.
