# LosantSync Configurations

Pre-defined LosantSync CR configurations for the losant-device controller.

| File | Interval | Use Case |
|---|---|---|
| `default.yaml` | 5 minutes | Standard demos |
| `high-frequency.yaml` | 1 minute | Live demos, real-time responsiveness |

To apply a configuration to a deployed cluster:

```bash
ldc-demo apply config default my-cluster
ldc-demo apply config high-frequency my-cluster
```

To add a custom configuration, create a new YAML file in this directory following
the same `LosantSync` CR schema and commit it to the repository.
