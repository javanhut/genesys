#!/bin/bash
# Bash completion script for genesys

_genesys_completions() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"

    # Main commands
    local commands="interact execute discover config version help"
    
    # Provider options
    local providers="aws gcp azure tencent"
    
    # Resource types for interact
    local resource_types="S3_Storage_Bucket Compute_Instance Database Function Network"
    
    # Config subcommands
    local config_commands="setup list show validate"

    case "${COMP_CWORD}" in
        1)
            # First level: main commands
            COMPREPLY=( $(compgen -W "${commands}" -- ${cur}) )
            return 0
            ;;
        2)
            case "${prev}" in
                interact)
                    # No direct completion, interactive mode
                    return 0
                    ;;
                execute)
                    # Complete with .yaml and .toml files
                    COMPREPLY=( $(compgen -f -X '!*.@(yaml|yml|toml)' -- ${cur}) )
                    return 0
                    ;;
                discover)
                    # Complete with provider names
                    COMPREPLY=( $(compgen -W "${providers}" -- ${cur}) )
                    return 0
                    ;;
                config)
                    # Complete with config subcommands
                    COMPREPLY=( $(compgen -W "${config_commands}" -- ${cur}) )
                    return 0
                    ;;
                *)
                    return 0
                    ;;
            esac
            ;;
        3)
            case "${COMP_WORDS[1]}" in
                execute)
                    # Options for execute command
                    local execute_opts="--dry-run --force --output --parallel"
                    COMPREPLY=( $(compgen -W "${execute_opts}" -- ${cur}) )
                    return 0
                    ;;
                discover)
                    # Resource types for discover
                    local discover_resources="compute storage network database serverless all"
                    if [[ " ${providers} " =~ " ${prev} " ]]; then
                        COMPREPLY=( $(compgen -W "${discover_resources}" -- ${cur}) )
                    fi
                    return 0
                    ;;
                config)
                    case "${prev}" in
                        setup)
                            # Provider options for config setup
                            COMPREPLY=( $(compgen -W "${providers}" -- ${cur}) )
                            return 0
                            ;;
                        show|validate)
                            # Provider options for show/validate
                            COMPREPLY=( $(compgen -W "${providers}" -- ${cur}) )
                            return 0
                            ;;
                    esac
                    ;;
            esac
            ;;
        *)
            # Handle flags at any position
            case "${cur}" in
                -*)
                    # Global flags
                    local global_flags="--help -h --version -v --verbose --debug"
                    
                    # Command-specific flags
                    case "${COMP_WORDS[1]}" in
                        execute)
                            local execute_flags="--dry-run --force --output -o --parallel -p"
                            COMPREPLY=( $(compgen -W "${global_flags} ${execute_flags}" -- ${cur}) )
                            ;;
                        discover)
                            local discover_flags="--output -o --format -f --filter"
                            COMPREPLY=( $(compgen -W "${global_flags} ${discover_flags}" -- ${cur}) )
                            ;;
                        config)
                            local config_flags="--global --show-path"
                            COMPREPLY=( $(compgen -W "${global_flags} ${config_flags}" -- ${cur}) )
                            ;;
                        *)
                            COMPREPLY=( $(compgen -W "${global_flags}" -- ${cur}) )
                            ;;
                    esac
                    return 0
                    ;;
            esac
            
            # File completion for paths
            if [[ "${prev}" == "--output" ]] || [[ "${prev}" == "-o" ]]; then
                COMPREPLY=( $(compgen -f -- ${cur}) )
                return 0
            fi
            ;;
    esac
}

# Register the completion function
complete -F _genesys_completions genesys