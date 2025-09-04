# Fish completion script for genesys

# Main commands
complete -c genesys -n __fish_use_subcommand -a interact -d "Interactive resource creation wizard"
complete -c genesys -n __fish_use_subcommand -a execute -d "Execute a configuration file"
complete -c genesys -n __fish_use_subcommand -a discover -d "Discover existing cloud resources"
complete -c genesys -n __fish_use_subcommand -a config -d "Manage Genesys configuration"
complete -c genesys -n __fish_use_subcommand -a version -d "Show version information"
complete -c genesys -n __fish_use_subcommand -a help -d "Show help information"

# Global flags
complete -c genesys -s h -l help -d "Show help"
complete -c genesys -s v -l version -d "Show version"
complete -c genesys -l verbose -d "Enable verbose output"
complete -c genesys -l debug -d "Enable debug output"

# Execute command
complete -c genesys -n "__fish_seen_subcommand_from execute" -F -r -d "Configuration file" -a "*.yaml *.yml *.toml"
complete -c genesys -n "__fish_seen_subcommand_from execute" -l dry-run -d "Preview changes without applying"
complete -c genesys -n "__fish_seen_subcommand_from execute" -l force -d "Force execution without confirmation"
complete -c genesys -n "__fish_seen_subcommand_from execute" -s o -l output -x -a "json yaml table" -d "Output format"
complete -c genesys -n "__fish_seen_subcommand_from execute" -s p -l parallel -d "Execute resources in parallel"

# Discover command
complete -c genesys -n "__fish_seen_subcommand_from discover; and not __fish_seen_subcommand_from aws gcp azure tencent" -a "aws gcp azure tencent" -d "Cloud provider"
complete -c genesys -n "__fish_seen_subcommand_from discover; and __fish_seen_subcommand_from aws gcp azure tencent" -a "compute storage network database serverless all" -d "Resource type"
complete -c genesys -n "__fish_seen_subcommand_from discover" -s o -l output -r -d "Output file"
complete -c genesys -n "__fish_seen_subcommand_from discover" -s f -l format -x -a "json yaml table" -d "Output format"
complete -c genesys -n "__fish_seen_subcommand_from discover" -l filter -x -d "Filter resources"

# Config command
complete -c genesys -n "__fish_seen_subcommand_from config; and not __fish_seen_subcommand_from setup list show validate" -a setup -d "Setup provider configuration"
complete -c genesys -n "__fish_seen_subcommand_from config; and not __fish_seen_subcommand_from setup list show validate" -a list -d "List configured providers"
complete -c genesys -n "__fish_seen_subcommand_from config; and not __fish_seen_subcommand_from setup list show validate" -a show -d "Show provider configuration"
complete -c genesys -n "__fish_seen_subcommand_from config; and not __fish_seen_subcommand_from setup list show validate" -a validate -d "Validate provider configuration"

# Config subcommands with providers
complete -c genesys -n "__fish_seen_subcommand_from config; and __fish_seen_subcommand_from setup show validate" -a "aws gcp azure tencent" -d "Cloud provider"
complete -c genesys -n "__fish_seen_subcommand_from config" -l global -d "Use global configuration"
complete -c genesys -n "__fish_seen_subcommand_from config" -l show-path -d "Show configuration file path"