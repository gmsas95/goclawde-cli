# Config Package

Configuration management using Viper for:
- YAML/JSON/TOML config files
- Environment variables (NANOBOT_* prefix)
- Command line flags
- Sensible defaults

Priority (highest to lowest):
1. CLI flags (`--data`, `--config`)
2. Environment variables
3. Config file
4. Defaults
