# IBM Fusion MCP Server - Operations Runbook

**Canonical reference for managing the IBM Fusion fork of kubernetes-mcp-server**

**Maintainer:** Sandeep Bazar
**GitHub:** [@sandeepbazar](https://github.com/sandeepbazar)
**Repository:** [ibm-fusion-mcp-server](https://github.com/sandeepbazar/ibm-fusion-mcp-server)
**Upstream:** [containers/kubernetes-mcp-server](https://github.com/containers/kubernetes-mcp-server)

## What is This Repository?

This is **ibm-fusion-mcp-server**, a fork of [containers/kubernetes-mcp-server](https://github.com/containers/kubernetes-mcp-server) that adds IBM Fusion-specific MCP tool extensions with **multi-cluster and fleet management** capabilities.

When Fusion tools are disabled (the default), the server behaves identically to upstream.

---

## Prerequisites

| Requirement | Version | Notes |
|-------------|---------|-------|
| **Go** | 1.25+ | For building from source |
| **kubectl** | any | Configured with cluster access |
| **kubeconfig** | — | At `~/.kube/config` or via `KUBECONFIG` env var |
| **Node.js** | 18+ | Only needed for MCP Inspector testing |

---

## Quick Start

### 1. Clone and Build

```bash
git clone https://github.com/sandeepbazar/ibm-fusion-mcp-server.git
cd ibm-fusion-mcp-server
make build
```

### 2. Run the Server

Two things are required to activate Fusion tools:

1. Set the environment variable `FUSION_TOOLS_ENABLED=true`
2. Include `fusion` in the `--toolsets` flag

#### STDIO Mode (for MCP clients like Claude Desktop)

```bash
FUSION_TOOLS_ENABLED=true ./kubernetes-mcp-server --toolsets core,config,helm,fusion
```

#### HTTP Mode (for MCP Inspector, remote access, or multi-client)

```bash
FUSION_TOOLS_ENABLED=true ./kubernetes-mcp-server \
  --port 9900 \
  --toolsets core,config,helm,fusion
```

The server starts on `http://localhost:9900` with endpoints:
- `/mcp` - Streamable HTTP MCP endpoint
- `/sse` - SSE transport endpoint
- `/healthz` - Health check
- `/stats` - Runtime statistics
- `/metrics` - Prometheus metrics

### 3. Verify Fusion Tools Are Loaded

```bash
# HTTP mode - check the log output for this line:
#   Toolsets: core, config, helm, fusion

# Or hit the health endpoint:
curl -s http://localhost:9900/healthz
```

### 4. Test with MCP Inspector

```bash
FUSION_TOOLS_ENABLED=true \
  npx @modelcontextprotocol/inspector $(pwd)/kubernetes-mcp-server \
  --toolsets core,config,helm,fusion
```

Open the URL shown in the terminal and invoke a tool:
```json
{
  "name": "fusion.storage.summary",
  "arguments": {}
}
```

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `FUSION_TOOLS_ENABLED` | `false` | **Required** - Set to `true` to enable Fusion tools |
| `KUBECONFIG` | `~/.kube/config` | Path to your kubeconfig file |
| `FUSION_TIMEOUT` | `30` | Operation timeout in seconds |
| `FUSION_LOG_BODY` | `none` | Diagnostic HTTP body logging: `none`, `summary`, or `full`. Requires `--log-level 6` or higher to produce output |

### Diagnostic Logging

The `FUSION_LOG_BODY` variable enables response body diagnostics for Fusion cluster requests. This is useful for debugging API responses.

```bash
# Summary mode - logs method, URL, status, kind, item count
FUSION_TOOLS_ENABLED=true FUSION_LOG_BODY=summary \
  ./kubernetes-mcp-server --port 9900 --log-level 6 \
  --toolsets core,config,helm,fusion

# Full mode - also logs pretty-printed JSON body (capped at 16KB)
FUSION_TOOLS_ENABLED=true FUSION_LOG_BODY=full \
  ./kubernetes-mcp-server --port 9900 --log-level 6 \
  --toolsets core,config,helm,fusion
```

All Fusion cluster requests use JSON wire format (never protobuf), so logs are always human-readable.

---

## Tool Catalog

All 11 tools support multi-cluster targeting.

| Tool Name | Domain | Description |
|-----------|--------|-------------|
| `fusion.storage.summary` | Storage | Storage classes, PVC stats, ODF detection |
| `fusion.datafoundation.status` | Data Foundation | ODF/OCS installation and health status |
| `fusion.gdp.status` | Global Data Platform | IBM Spectrum Scale/GDP status |
| `fusion.backup.jobs.list` | Backup & Restore | List OADP/Velero backup jobs |
| `fusion.dr.status` | Disaster Recovery | Metro/Regional DR status |
| `fusion.catalog.status` | Data Cataloging | Data catalog service status |
| `fusion.cas.status` | Content Aware Storage | CAS deployment status |
| `fusion.serviceability.summary` | Serviceability | Must-gather and logging status |
| `fusion.observability.summary` | Observability | Prometheus, Grafana, OTEL status |
| `fusion.virtualization.status` | Virtualization | KubeVirt/OpenShift Virt status |
| `fusion.hcp.status` | Hosted Control Planes | HyperShift/HCP status |

---

## Multi-Cluster Targeting

### Targeting Model

| Target Type | Description | Use Case |
|-------------|-------------|----------|
| **single** | One specific cluster (default) | Most common operations |
| **multi** | Explicitly named clusters | Coordinated cross-cluster operations |
| **fleet** | All clusters in the fleet | Fleet-wide health checks |
| **selector** | Clusters matching labels | Environment-based targeting (prod, dev) |
| **all** | All registered clusters | Global operations |

### Multi-Cluster Setup

The server automatically registers all contexts from your kubeconfig:

```bash
# List your available contexts
kubectl config get-contexts

# All contexts are registered automatically when the server starts
```

---

## Usage Examples

### Default (Single Cluster - Current Context)

When no `target` is specified, tools use the current kubeconfig context:

```json
{
  "name": "fusion.datafoundation.status",
  "arguments": {}
}
```

### Single Cluster (Named Context)

```json
{
  "name": "fusion.backup.jobs.list",
  "arguments": {
    "target": {
      "type": "single",
      "cluster": "prod-us-east-1"
    }
  }
}
```

### Multi-Cluster (Explicit List)

```json
{
  "name": "fusion.dr.status",
  "arguments": {
    "target": {
      "type": "multi",
      "clusters": ["prod-us-east-1", "prod-us-west-2"]
    }
  }
}
```

### Fleet (All Clusters)

```json
{
  "name": "fusion.virtualization.status",
  "arguments": {
    "target": {
      "type": "fleet"
    }
  }
}
```

### Selector (Label-Based)

```json
{
  "name": "fusion.observability.summary",
  "arguments": {
    "target": {
      "type": "selector",
      "selector": "env=prod,region=us"
    }
  }
}
```

### Response Format

All tools return a consistent response structure:

```json
{
  "target": {
    "type": "single"
  },
  "clusterResults": {
    "default": {
      "clusterName": "default",
      "success": true,
      "data": {
        "installed": true,
        "ready": true,
        "namespace": "openshift-storage",
        "storageClasses": ["ocs-storagecluster-ceph-rbd"]
      }
    }
  },
  "summary": {
    "clustersTotal": 1,
    "clustersOk": 1,
    "clustersFailed": 0
  }
}
```

---

## Fleet Admin Scenarios

### Morning Health Check - Data Foundation Across All Clusters

```json
{
  "name": "fusion.datafoundation.status",
  "arguments": { "target": {"type": "fleet"} }
}
```

### Verify Backup Jobs on Primary and DR Sites

```json
{
  "name": "fusion.backup.jobs.list",
  "arguments": {
    "target": {
      "type": "multi",
      "clusters": ["prod-primary", "prod-dr"]
    }
  }
}
```

### Inventory Check - Which Clusters Have Virtualization

```json
{
  "name": "fusion.virtualization.status",
  "arguments": { "target": {"type": "fleet"} }
}
```

### DR Readiness Audit

```json
{
  "name": "fusion.dr.status",
  "arguments": { "target": {"type": "fleet"} }
}
```

### Compare Storage Classes Across Prod Clusters

```json
{
  "name": "fusion.storage.summary",
  "arguments": {
    "target": {
      "type": "multi",
      "clusters": ["prod-1", "prod-2", "prod-3"]
    }
  }
}
```

### Observability Stack Status Fleet-Wide

```json
{
  "name": "fusion.observability.summary",
  "arguments": { "target": {"type": "fleet"} }
}
```

---

## Architecture

### IBM Fusion Domain Mapping

IBM Fusion is organized into service domains that this MCP server covers:

**Fusion Data Services:**
- **Data Foundation** - OpenShift Data Foundation (ODF/OCS) for persistent storage
- **Global Data Platform (GDP)** - IBM Spectrum Scale for high-performance file storage
- **Backup & Restore** - OADP/Velero-based backup and disaster recovery
- **Disaster Recovery** - Metro DR and Regional DR for business continuity
- **Data Cataloging** - Metadata management and data discovery
- **Content Aware Storage (CAS)** - Intelligent storage tiering

**Fusion Base:**
- **Serviceability** - Must-gather, logging, diagnostics
- **Observability** - Prometheus, Grafana, OpenTelemetry

**Additional Capabilities:**
- **Virtualization** - KubeVirt/OpenShift Virtualization for VM workloads
- **Hosted Control Planes (HCP)** - HyperShift for multi-tenant cluster management

### Directory Structure

```
ibm-fusion-mcp-server/
├── README.md                              # Upstream README
├── README.FUSION.md                       # This file - canonical Fusion reference
│
├── internal/fusion/                       # Internal Fusion implementation
│   ├── config/
│   │   ├── config.go                     # Feature gate (FUSION_TOOLS_ENABLED)
│   │   └── config_test.go
│   ├── clients/
│   │   ├── kubernetes.go                 # K8s client wrappers
│   │   ├── registry.go                   # Multi-cluster client registry
│   │   └── diagnostic_round_tripper.go   # HTTP diagnostic logging (FUSION_LOG_BODY)
│   ├── services/
│   │   ├── common.go                     # Shared service utilities
│   │   ├── storage.go                    # Storage domain logic
│   │   ├── datafoundation.go            # Data Foundation logic
│   │   ├── backup.go                     # Backup & Restore logic
│   │   └── multidom.go                   # Multi-domain services
│   └── targeting/
│       └── target.go                     # Multi-cluster targeting model
│
├── pkg/toolsets/fusion/                   # Public Fusion toolset API
│   ├── registry.go                       # Toolset registration
│   ├── toolset.go                        # Toolset implementation
│   ├── storage/
│   │   └── tool_storage_summary.go
│   ├── datafoundation/
│   │   └── tool_status.go
│   ├── backup/
│   │   └── tool_jobs_list.go
│   └── alltools/
│       └── tools.go                      # All other domain tools
│
├── docs/fusion/
│   └── README.md                         # Quick start guide
│
└── [upstream files...]
```

### Design Goals

1. **Keep upstream clean** - Minimize modifications to upstream code
2. **Isolate Fusion changes** - All Fusion code in `internal/fusion/` and `pkg/toolsets/fusion/`
3. **Feature gating** - Disabled by default via `FUSION_TOOLS_ENABLED`
4. **Multi-cluster support** - Single, multi, fleet, and selector targeting
5. **Maintain sync-ability** - Regular upstream syncs with minimal conflicts
6. **JSON wire format** - All Fusion API calls use JSON (never protobuf) for readable diagnostics

### Integration Points (Upstream Modifications)

We touch **exactly 2 upstream files**:

**1. `pkg/toolsets/toolsets.go`** - Hook for Fusion toolset registration:
```go
func init() {
    registerFusionTools()
}
var registerFusionTools = func() {}
func SetFusionRegistration(fn func()) {
    registerFusionTools = fn
}
```

**2. `pkg/mcp/modules.go`** - Blank import to trigger Fusion init:
```go
_ "github.com/containers/kubernetes-mcp-server/pkg/toolsets/fusion"
```

The Fusion `init()` in `pkg/toolsets/fusion/registry.go` calls `RegisterTools()` directly (since `toolsets.init()` runs first due to Go init ordering).

---

## Testing

```bash
# Run all Fusion tests
go test ./internal/fusion/... ./pkg/toolsets/fusion/...

# With coverage
go test -cover ./internal/fusion/... ./pkg/toolsets/fusion/...

# Full project build + lint
make build
```

---

## Troubleshooting

### Fusion Tools Not Appearing

**Symptom:** Server starts but Fusion tools are not available.

**Checklist:**
1. Verify `FUSION_TOOLS_ENABLED=true` is set
2. Verify `--toolsets` includes `fusion` (e.g., `--toolsets core,config,helm,fusion`)
3. Server log shows `Toolsets: core, config, helm, fusion`
4. Rebuild after changes: `make build`

```bash
# Quick verification
FUSION_TOOLS_ENABLED=true ./kubernetes-mcp-server \
  --port 9900 --log-level 2 \
  --toolsets core,config,helm,fusion
# Look for: "Toolsets: core, config, helm, fusion" in the log output
```

### Component Shows "Not Installed"

**Symptom:** Tool returns `"installed": false` but component exists.

**Causes:**
- Namespace mismatch - component in a different namespace than expected
- CRD not found - Custom Resource Definition not installed
- Insufficient RBAC permissions

**Debug:**
```bash
kubectl get crds | grep <component-name>
kubectl get ns | grep <namespace-name>
kubectl auth can-i list <resource> --as=system:serviceaccount:<ns>:<sa>
```

### Multi-Cluster Operation Timeout

**Symptom:** Operations timeout when targeting multiple clusters.

**Solution:** Increase timeout in the target:
```json
{
  "target": {
    "type": "multi",
    "clusters": ["cluster1", "cluster2"],
    "timeout": 60
  }
}
```

Also verify each cluster is reachable:
```bash
kubectl --context=<context-name> get nodes
```

### Build Error: Module Cache Corruption

**Symptom:**
```
go: module ... found but does not contain package ...
```

**Solution:**
```bash
go clean -modcache
go mod download
make build
```

### Connection Refused / Cluster Unreachable

```bash
# Verify kubeconfig is valid
kubectl config view
kubectl config get-contexts

# Test each context
kubectl --context=<context-name> get nodes
```

---

## Upstream Sync

```bash
# Quick sync
./hack/fusion/sync.sh
```

All Fusion code is isolated in `internal/fusion/` and `pkg/toolsets/fusion/`, so upstream syncs should produce minimal conflicts. The only upstream files touched are `pkg/toolsets/toolsets.go` and `pkg/mcp/modules.go`.

---

## Planned Enhancements

1. **Storage** - `fusion.storage.pvc.list`, `fusion.storage.pvc.resize`, `fusion.storage.classes.compare`
2. **Backup** - `fusion.backup.policies.list`, `fusion.backup.schedule.status`
3. **Virtualization** - `fusion.vm.list`, `fusion.vm.migrate`
4. **HCP** - `fusion.hcp.list`, `fusion.hcp.nodepool.status`

---

## Support

**Maintainer:** Sandeep Bazar
**GitHub:** [@sandeepbazar](https://github.com/sandeepbazar)

For issues: https://github.com/sandeepbazar/ibm-fusion-mcp-server/issues
